package segment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

// fakeDocDB 是 common.DocumentDB 的最小假实现：只统计落库调用、并按 _id 模拟 upsert
// （足够覆盖「重复命中只更新、不插新行」的台账语义），不连任何真实 mongo。
type fakeDocDB struct {
	insertCalls int
	updateCalls int

	insertErr error
	updateErr error

	// rows 以 _id 为键模拟 upsert 后的文档状态。
	rows map[string]map[string]interface{}
}

func newFakeDocDB() *fakeDocDB {
	return &fakeDocDB{rows: map[string]map[string]interface{}{}}
}

func (f *fakeDocDB) Insert(_ context.Context, _ string, docs []interface{}) (int, error) {
	f.insertCalls++
	if f.insertErr != nil {
		return 0, f.insertErr
	}
	return len(docs), nil
}

func (f *fakeDocDB) Find(_ context.Context, _ string, _ interface{}, _ ...*options.FindOptions) (*mongo.Cursor, error) {
	return nil, nil
}

func (f *fakeDocDB) Update(_ context.Context, _ string, filter, docs interface{}) (*mongo.UpdateResult, error) {
	f.updateCalls++
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	id := fakeFilterID(filter)
	if id != "" {
		row := f.rows[id]
		if row == nil {
			row = map[string]interface{}{}
			f.rows[id] = row
		}
		m, _ := docs.(bson.M)
		if soi, ok := m["$setOnInsert"].(bson.M); ok {
			for k, v := range soi {
				if _, exists := row[k]; !exists {
					row[k] = v
				}
			}
		}
		if set, ok := m["$set"].(bson.M); ok {
			for k, v := range set {
				row[k] = v
			}
		}
		if inc, ok := m["$inc"].(bson.M); ok {
			for k, v := range inc {
				n, _ := v.(int)
				cur, _ := row[k].(int)
				row[k] = cur + n
			}
		}
	}

	return &mongo.UpdateResult{}, nil
}

func (f *fakeDocDB) Delete(_ context.Context, _ string, _ interface{}) (int, error) { return 0, nil }

func (f *fakeDocDB) Aggregate(_ context.Context, _ string, _ interface{}, _ interface{}) error {
	return nil
}

func (f *fakeDocDB) FindOneAndUpdate(_ context.Context, _ string, _ interface{}, _ interface{}) error {
	return nil
}

func (f *fakeDocDB) CountDocuments(_ context.Context, _ string, _ interface{}) (int64, error) {
	return 0, nil
}

func (f *fakeDocDB) FindOne(_ context.Context, _ string, _ interface{}, _ ...*options.FindOneOptions) *mongo.SingleResult {
	return nil
}

func fakeFilterID(filter interface{}) string {
	switch v := filter.(type) {
	case bson.M:
		if s, ok := v["_id"].(string); ok {
			return s
		}
	case bson.D:
		for _, e := range v {
			if e.Key == "_id" {
				if s, ok := e.Value.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

// 台账去重：同一 (from,to) 重复命中只更新 last_seen/attempts，不插新行。
func TestRecordSkippedPart_DedupesByRange(t *testing.T) {
	fake := newFakeDocDB()
	s := &Segment{db: fake}
	l := zap.NewNop().Sugar()
	pf := PartFailure{
		FromEpoch: 100,
		ToEpoch:   115,
		Class:     FailureClassState,
		ErrDigest: "digest",
		CID:       testMissingStateCID,
		Attempts:  1,
	}

	if err := s.recordSkippedPart(context.Background(), l, pf); err != nil {
		t.Fatalf("第一次登记失败: %v", err)
	}
	if err := s.recordSkippedPart(context.Background(), l, pf); err != nil {
		t.Fatalf("第二次登记失败: %v", err)
	}

	if fake.updateCalls != 2 {
		t.Fatalf("期望 2 次 upsert，实际 %d", fake.updateCalls)
	}
	if len(fake.rows) != 1 {
		t.Fatalf("同一区间重复命中不应插新行，实际行数 %d", len(fake.rows))
	}
	id := model.SkippedPartID(100, 115)
	row := fake.rows[id]
	if row == nil {
		t.Fatalf("台账未写入 _id=%s", id)
	}
	if got := row["attempts"]; got != 2 {
		t.Fatalf("attempts 应累加到 2，实际 %v", got)
	}
	if _, ok := row["first_seen"]; !ok {
		t.Fatalf("应写入 first_seen")
	}
	if _, ok := row["last_seen"]; !ok {
		t.Fatalf("应写入 last_seen")
	}
}

// 写库不可用：必须报错（铁律 —— 台账写不进去就不允许跳过）。
func TestRecordSkippedPart_NilDBReturnsError(t *testing.T) {
	s := &Segment{}
	if err := s.recordSkippedPart(context.Background(), zap.NewNop().Sugar(),
		PartFailure{FromEpoch: 1, ToEpoch: 2, Class: FailureClassState}); err == nil {
		t.Fatal("写库为 nil 时必须报错")
	}
}

// 底层写失败：必须把错误透出（调用方据此按不可登记处理）。
func TestRecordSkippedPart_UpdateErrorPropagates(t *testing.T) {
	fake := newFakeDocDB()
	fake.updateErr = errors.New("mongo down")
	s := &Segment{db: fake}
	if err := s.recordSkippedPart(context.Background(), zap.NewNop().Sugar(),
		PartFailure{FromEpoch: 1, ToEpoch: 2, Class: FailureClassState}); err == nil {
		t.Fatal("底层写失败必须透出 error")
	}
}

// fakeDoc 是最小的 common.Document 实现，供 persistPartDocs 的落库断言使用。
type fakeDoc struct{}

func (d *fakeDoc) CollectionName() string { return "FakeDoc" }
func (d *fakeDoc) EpochField() *string    { s := "Epoch"; return &s }
func (d *fakeDoc) ResetPolicy(_ *abi.ChainEpoch, _ *abi.ChainEpoch) (interface{}, bool) {
	return nil, false
}
func (d *fakeDoc) IsMutable() bool { return false }

// persist=false 分支必须完全不落库；persist=true 才落库。
// 这直接锁住「坏 part 之后的 part 只抽取、不落库」这一正确性约束（§8.3）。
func TestPersistPartDocs_PersistFalseSkipsWrites(t *testing.T) {
	fake := newFakeDocDB()
	s := &Segment{opts: DefaultOptions(), db: fake}
	pctx := &persistCtx{
		ctx:        context.Background(),
		log:        zap.NewNop().Sugar(),
		persistSem: make(chan struct{}, 2),
	}
	docs := [][]common.Document{{&fakeDoc{}}}

	// persist=false：不得有任何落库调用。
	if err := s.persistPartDocs(pctx, pctx.log, docs, false); err != nil {
		t.Fatalf("persist=false 不应报错: %v", err)
	}
	if fake.insertCalls != 0 || fake.updateCalls != 0 {
		t.Fatalf("persist=false 不得落库，实际 insert=%d update=%d", fake.insertCalls, fake.updateCalls)
	}

	// persist=true（异步路径）：必须落库。
	if err := s.persistPartDocs(pctx, pctx.log, docs, true); err != nil {
		t.Fatalf("persist=true 不应报错: %v", err)
	}
	if err := waitAsyncPersist(&pctx.asyncPersistWaitGroup, time.Second); err != nil {
		t.Fatalf("等待异步落库失败: %v", err)
	}
	if fake.insertCalls == 0 {
		t.Fatalf("persist=true 应至少落库一次，实际 insert=%d", fake.insertCalls)
	}
}
