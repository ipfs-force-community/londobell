package multiquery

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// 真实链路的内存对比测量（需要一个一次性 scratch mongod）：
//
//	MONGO_TEST_URI=mongodb://127.0.0.1:27099 go test ./cmd/londobell-api/multi-query/ \
//	    -run TestFanoutGatePeakMemoryAgainstMongo -v -timeout 10m
//
// 它走的是**真的** mongo 路径（真的 col.Aggregate + cur.All 全量物化），
// 用「多个库 × 多文档」把每个在飞查询的结果缓冲区做大，然后对比
// 「闸门关（= 改前）」与「闸门开（限 2）」在同等并发下的峰值 HeapInuse。
//
// 没设置 MONGO_TEST_URI 时整体跳过。
func TestFanoutGatePeakMemoryAgainstMongo(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("set MONGO_TEST_URI (scratch mongod, never a production/cold-storage node) to run this integration test")
	}

	const (
		libs     = 8
		requests = 10
		docs     = 4000
		docBytes = 400
	)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMaxPoolSize(64))
	if err != nil {
		t.Fatalf("connect %v: %v", uri, err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping %v: %v", uri, err)
	}

	// 建库/灌数据：每个库一张 ExecTrace，docs 条 ~docBytes 的文档。
	payload := strings.Repeat("x", docBytes)
	names := make([]string, 0, libs)
	for i := 0; i < libs; i++ {
		names = append(names, fmt.Sprintf("fanoutmem_lb_%02d", i))
	}

	defer func() {
		for _, n := range names {
			_ = client.Database(n).Drop(context.Background())
		}
	}()

	for _, n := range names {
		col := client.Database(n).Collection("ExecTrace")
		_ = client.Database(n).Drop(ctx)

		batch := make([]interface{}, 0, 1000)
		for d := 0; d < docs; d++ {
			batch = append(batch, bson.M{"IsBlock": true, "Epoch": d, "Blob": payload})
			if len(batch) == 1000 {
				if _, err := col.InsertMany(ctx, batch); err != nil {
					t.Fatalf("insert into %v: %v", n, err)
				}
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			if _, err := col.InsertMany(ctx, batch); err != nil {
				t.Fatalf("insert into %v: %v", n, err)
			}
		}
	}

	countLists := make([]CountUtil, 0, libs)
	for _, n := range names {
		db := client.Database(n)
		countLists = append(countLists, CountUtil{
			Cols:  common.Collections{DB: db, Cols: []*mongo.Collection{db.Collection("ExecTrace")}},
			DType: smodel.Cold,
		})
	}

	pipe, err := util.Parse(map[string]interface{}{}, `[{"$match": {"IsBlock": true}}]`)
	if err != nil {
		t.Fatalf("parse pipeline: %v", err)
	}

	measure := func(t *testing.T, gateLimit int) (peakInflight int, peakHeapInuse uint64, gotDocs int) {
		t.Helper()

		useGate(t, gateLimit, 30*time.Second)
		resetShardGateStats()

		runtime.GC()

		stop := make(chan struct{})
		var (
			swg     sync.WaitGroup
			heapMax uint64
		)
		swg.Add(1)
		go func() {
			defer swg.Done()
			var ms runtime.MemStats
			for {
				select {
				case <-stop:
					return
				default:
				}

				runtime.ReadMemStats(&ms)
				if ms.HeapInuse > heapMax {
					heapMax = ms.HeapInuse
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()

		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			docs int
		)
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := MultiTraversalQuery(context.Background(), pipe, countLists, "ExecTrace")
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				mu.Lock()
				docs = len(res)
				mu.Unlock()
			}()
		}
		wg.Wait()

		close(stop)
		swg.Wait()

		// 由闸门自身记录的在飞峰值（跨本轮所有查询）。
		stats := ShardGateSnapshot()

		return int(stats.Peak), heapMax, docs
	}

	// 先测「闸门开」，再测「闸门关」（避免上一轮遗留堆抬高这一轮）。
	gInflight, gHeap, gDocs := measure(t, 2)
	uInflight, uHeap, uDocs := measure(t, 0)

	t.Logf("per-query result size ≈ %.1f MiB (docs=%d)", float64(docs)*float64(docBytes)/(1<<20), docs)
	t.Logf("gate=2 : peak_inflight=%d peak_heap_inuse=%.1f MiB result_docs=%d", gInflight, float64(gHeap)/(1<<20), gDocs)
	t.Logf("gate=off: peak_inflight=%d peak_heap_inuse=%.1f MiB result_docs=%d", uInflight, float64(uHeap)/(1<<20), uDocs)

	if uInflight <= gInflight {
		t.Fatalf("ungated peak in-flight (%d) must exceed gated (%d)", uInflight, gInflight)
	}
	if gHeap >= uHeap {
		t.Fatalf("gate did not reduce peak HeapInuse: gated=%.1fMiB ungated=%.1fMiB",
			float64(gHeap)/(1<<20), float64(uHeap)/(1<<20))
	}
	// 结果集一致：闸门不改变返回值。
	if gDocs != uDocs {
		t.Fatalf("result size changed with the gate: gated=%d ungated=%d", gDocs, uDocs)
	}
	if gDocs == 0 {
		t.Fatalf("pipeline returned no docs; measurement is meaningless")
	}
}
