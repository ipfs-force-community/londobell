package aggregators

import (
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
)

// mpoolTestSnapshot 造一批 nonce 递增的消息（nonce 不同 ⇒ Message.Cid 不同），
// 包成 getUnStoredMsgs 的返回形状，用来驱动 refreshMpoolCache。
func mpoolTestSnapshot(start, n int) map[*types.TipSet][]*types.SignedMessage {
	from, err := address.NewIDAddress(1)
	if err != nil {
		panic(err)
	}
	msgs := make([]*types.SignedMessage, 0, n)
	for i := start; i < start+n; i++ {
		msgs = append(msgs, &types.SignedMessage{
			Message:   types.Message{From: from, To: from, Nonce: uint64(i)},
			Signature: crypto.Signature{Type: crypto.SigTypeBLS},
		})
	}
	return map[*types.TipSet][]*types.SignedMessage{{}: msgs}
}

// 性质一：refreshMpoolCache 必须可并发调用 —— 生产里 1s ticker（RefreshMpool）
// 与 HTTP handler（GetMpool）会同时进这个函数，任何一方无锁读写包级 poolCache
// 都会让 Go 运行时抛 `fatal error: concurrent map read and map write` 并 exit 2。
// -race 下必须干净（本测试在修复前会 RED）。
func TestRefreshMpoolCacheConcurrent(t *testing.T) {
	snap := mpoolTestSnapshot(0, 200)
	readers := mpoolTestSnapshot(0, 200)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				refreshMpoolCache(snap)
			}
		}()
	}
	// 读侧也必须能并发（handler 在返回 pending 列表时要读每条的 MsgTime）
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				for _, msgs := range readers {
					for _, m := range msgs {
						poolCacheAt(m.Cid())
					}
				}
			}
		}()
	}
	wg.Wait()
}

// 性质二：时间戳语义不变 —— 同一 CID 只在首次出现时记时间，后续刷新必须沿用，
// 新出现的 CID 才拿当前时间。
func TestRefreshMpoolCacheKeepsFirstSeenTimestamp(t *testing.T) {
	first := mpoolTestSnapshot(0, 3)
	refreshMpoolCache(first)

	before := make(map[string]int64)
	for _, msgs := range first {
		for _, m := range msgs {
			before[m.Cid().String()] = poolCacheAt(m.Cid())
		}
	}
	if len(before) != 3 {
		t.Fatalf("expect 3 entries in poolCache, got %d", len(before))
	}

	time.Sleep(1100 * time.Millisecond)
	// 同一批 + 一条新消息
	second := mpoolTestSnapshot(0, 4)
	refreshMpoolCache(second)

	for _, msgs := range second {
		for _, m := range msgs {
			got := poolCacheAt(m.Cid())
			if old, ok := before[m.Cid().String()]; ok {
				if got != old {
					t.Fatalf("existing cid %s timestamp changed: %d -> %d", m.Cid(), old, got)
				}
				continue
			}
			if got <= 0 {
				t.Fatalf("new cid %s should get a fresh timestamp, got %d", m.Cid(), got)
			}
		}
	}
}
