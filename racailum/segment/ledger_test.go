package segment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	logging "github.com/ipfs/go-log/v2"
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
	part := mkTSS(t, 10, 11)

	// persist=false：不得有任何落库调用。
	if err := s.persistPartDocs(pctx, pctx.log, part, docs, false, nil); err != nil {
		t.Fatalf("persist=false 不应报错: %v", err)
	}
	if fake.insertCalls != 0 || fake.updateCalls != 0 {
		t.Fatalf("persist=false 不得落库，实际 insert=%d update=%d", fake.insertCalls, fake.updateCalls)
	}

	// persist=true（异步路径）：必须落库。
	if err := s.persistPartDocs(pctx, pctx.log, part, docs, true, nil); err != nil {
		t.Fatalf("persist=true 不应报错: %v", err)
	}
	if fails := pctx.asyncPersist.wait(time.Second); len(fails) != 0 {
		t.Fatalf("等待异步落库不应有失败归属: %+v", fails)
	}
	if fake.insertCalls == 0 {
		t.Fatalf("persist=true 应至少落库一次，实际 insert=%d", fake.insertCalls)
	}
}

// ---- 台账 heal 闭环（ReconcileHealedParts）单测 ----
//
// 形状（本缺陷修复后）：ReconcileHealedParts 不再发「无等值条件的一条 updateMany」
// （底层 DocumentDB.Update 恒带 Upsert=true，无匹配即 upsert 出一行垃圾台账，稳态下每轮 +1 行），
// 而是「先查候选 → 按 _id 逐个 $set」。候选查询走包级注入点 findHealCandidates——
// 单测据此注入可控候选集与错误，无需连真 mongo、也无需造 *mongo.Cursor
// （mongo-driver v1.5.0 无 NewCursorFromDocuments，造不出可被 cursor.All 解码的游标）。
//
// reconcileFakeDocDB 是台账假实现：
//   - 作为 DocumentDB：Update 按 _id 逐行记账（记录调用次数 / 被更新 id / 更新文档）；
//   - findCandidates：注入到 findHealCandidates 的假查询，按 mongo 语义求值 predicate
//     （{healed_at: null, to_epoch: {$lte: w}, error_class: {$in: [...]}}) 后返回候选行。

type reconcileFakeDocDB struct {
	rows map[string]map[string]interface{} // _id -> 字段

	updateCalls    int
	updatedIDs     []string // 每次 Update 命中的 _id（按调用顺序）
	lastUpdateDocs bson.M   // 最近一次 Update 的更新文档（断言 $set / $setOnInsert）
	updateErr      error

	findFilter bson.M // 最近一次候选查询收到的 predicate（未查询时保持 nil）
	findLimit  int64
	findErr    error
}

func newReconcileFake() *reconcileFakeDocDB {
	return &reconcileFakeDocDB{rows: map[string]map[string]interface{}{}}
}

// seedHole 塞入一行台账。[firstHealedAt] 非空时带上那个首次 heal 时间（用于断言不被覆盖）。
func (f *reconcileFakeDocDB) seedHole(from, to abi.ChainEpoch, class string, firstHealedAt *time.Time) {
	row := map[string]interface{}{
		"from_epoch":  from,
		"to_epoch":    to,
		"error_class": class,
	}
	if firstHealedAt != nil {
		row["healed_at"] = firstHealedAt
	} else {
		row["healed_at"] = nil
	}
	f.rows[model.SkippedPartID(from, to)] = row
}

// seedHoleNoHealedField 塞入一行「完全没有 healed_at 字段」的台账（模拟字段缺失，仍应视为未修复）。
func (f *reconcileFakeDocDB) seedHoleNoHealedField(from, to abi.ChainEpoch, class string) {
	f.seedHole(from, to, class, nil)
	delete(f.rows[model.SkippedPartID(from, to)], "healed_at")
}

// findCandidates 注入到 findHealCandidates：记录 predicate，按 predicate 求值后返回候选行。
func (f *reconcileFakeDocDB) findCandidates(_ context.Context, _ common.DocumentDB, filter bson.M, limit int64) ([]model.SkippedPart, error) {
	f.findFilter = filter
	f.findLimit = limit
	if f.findErr != nil {
		return nil, f.findErr
	}
	var out []model.SkippedPart
	for id, row := range f.rows {
		if !matchReconcileFilter(row, filter) {
			continue
		}
		from, _ := row["from_epoch"].(abi.ChainEpoch)
		to, _ := row["to_epoch"].(abi.ChainEpoch)
		class, _ := row["error_class"].(string)
		out = append(out, model.SkippedPart{ID: id, FromEpoch: from, ToEpoch: to, ErrorClass: class})
	}
	return out, nil
}

// Update：新形状下 filter 恒为 {_id: <id>}，逐行落账。若 _id 不在（竞态下 upsert 插行），
// 用 $setOnInsert 补全台账键——这样插进去的也是完整台账行，而不是只有 _id+healed_at 的垃圾行。
func (f *reconcileFakeDocDB) Update(_ context.Context, _ string, filter, docs interface{}) (*mongo.UpdateResult, error) {
	f.updateCalls++
	f.lastUpdateDocs, _ = docs.(bson.M)
	id := fakeFilterID(filter)
	f.updatedIDs = append(f.updatedIDs, id)
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	um, _ := docs.(bson.M)
	row := f.rows[id]
	if row == nil {
		row = map[string]interface{}{}
		f.rows[id] = row
		if soi, ok := um["$setOnInsert"].(bson.M); ok {
			for k, v := range soi {
				if _, exists := row[k]; !exists {
					row[k] = v
				}
			}
		}
	}
	if set, ok := um["$set"].(bson.M); ok {
		for k, v := range set {
			row[k] = v
		}
	}
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

// matchReconcileFilter 求值 heal 对账用到的 filter 形状：等值（含 null）、$lte、$in。
func matchReconcileFilter(row map[string]interface{}, filter bson.M) bool {
	for k, cond := range filter {
		val := row[k]
		switch c := cond.(type) {
		case nil:
			// {field: null} 命中「值为 null」或「字段缺失」——即未修复。
			if val != nil {
				return false
			}
		case bson.M:
			for op, ov := range c {
				switch op {
				case "$lte":
					lhs, _ := val.(abi.ChainEpoch)
					rhs, _ := ov.(abi.ChainEpoch)
					if int64(lhs) > int64(rhs) {
						return false
					}
				case "$in":
					list, _ := ov.([]string)
					s, _ := val.(string)
					if !stringInList(s, list) {
						return false
					}
				default:
					return false
				}
			}
		default:
			if val != cond {
				return false
			}
		}
	}
	return true
}

func stringInList(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (f *reconcileFakeDocDB) Insert(context.Context, string, []interface{}) (int, error) {
	return 0, nil
}

// Find 是默认 findHealCandidates 会调到的底层方法，只用于「查询报错 / nil 游标」两条失败路径
// （候选集路径改走注入的 findCandidates，因为造不出可被 cursor.All 解码的游标）。
func (f *reconcileFakeDocDB) Find(context.Context, string, interface{}, ...*options.FindOptions) (*mongo.Cursor, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return nil, nil // nil 游标：默认实现应据此返回 error（与 RegisteredHolesInRange 同口径）
}
func (f *reconcileFakeDocDB) Delete(context.Context, string, interface{}) (int, error) { return 0, nil }
func (f *reconcileFakeDocDB) Aggregate(context.Context, string, interface{}, interface{}) error {
	return nil
}
func (f *reconcileFakeDocDB) FindOneAndUpdate(context.Context, string, interface{}, interface{}) error {
	return nil
}
func (f *reconcileFakeDocDB) CountDocuments(context.Context, string, interface{}) (int64, error) {
	return 0, nil
}
func (f *reconcileFakeDocDB) FindOne(context.Context, string, interface{}, ...*options.FindOneOptions) *mongo.SingleResult {
	return nil
}

// swapHealFinder 临时替换候选查询实现，测试结束自动还原（避免污染其它用例）。
func swapHealFinder(t *testing.T, fn healCandidateFinder) {
	t.Helper()
	orig := findHealCandidates
	findHealCandidates = fn
	t.Cleanup(func() { findHealCandidates = orig })
}

// captureSegmentLogs 临时把包日志换成可观测 logger，返回收集结果（测试结束自动还原）。
//
// 包日志 `log` 是 *go-log.ZapEventLogger（内嵌 *zap.SugaredLogger）；Infow/Warnw 由内嵌字段
// 直接提升，故这里用一个零值 ZapEventLogger 挂上指向 observer core 的 SugaredLogger 即可。
func captureSegmentLogs(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.InfoLevel)
	orig := log
	var replacement logging.ZapEventLogger
	replacement.SugaredLogger = *zap.New(core).Sugar()
	log = &replacement
	t.Cleanup(func() { log = orig })
	return logs
}

func healedAtOf(f *reconcileFakeDocDB, from, to abi.ChainEpoch) interface{} {
	row := f.rows[model.SkippedPartID(from, to)]
	if row == nil {
		return nil
	}
	return row["healed_at"]
}

// ★ 回归用例（直接钉住本次缺陷）：台账里没有任何「未修复且被水位越过」的候选时，
// ReconcileHealedParts 必须**零 Update 调用** —— 这正是旧实现会 upsert 插一行垃圾台账的路径。
// 覆盖两种「无候选」：台账为空（稳态最常见的形状）；台账有行但都已修复 / 水位未越过 / 类别不在册。
func TestReconcileHealedParts_NoCandidatesIssuesZeroUpdates(t *testing.T) {
	t.Run("台账为空", func(t *testing.T) {
		fake := newReconcileFake()
		swapHealFinder(t, fake.findCandidates)

		s := &Segment{db: fake}
		if err := s.ReconcileHealedParts(context.Background(), 150, []string{FailureClassState, FailureClassData}); err != nil {
			t.Fatalf("对账不应报错: %v", err)
		}
		if fake.updateCalls != 0 {
			t.Fatalf("无候选必须零写入，实际 %d 次 Update", fake.updateCalls)
		}
	})

	t.Run("有行但无候选", func(t *testing.T) {
		fake := newReconcileFake()
		first := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
		fake.seedHole(10, 20, FailureClassState, &first) // 已修复 ⇒ 非候选
		fake.seedHole(200, 210, FailureClassState, nil)  // 水位 150 未越过 210 ⇒ 非候选
		fake.seedHole(30, 40, FailureClassData, nil)     // 类别不在册（只背书 state）⇒ 非候选
		swapHealFinder(t, fake.findCandidates)

		s := &Segment{db: fake}
		if err := s.ReconcileHealedParts(context.Background(), 150, []string{FailureClassState}); err != nil {
			t.Fatalf("对账不应报错: %v", err)
		}
		if fake.updateCalls != 0 {
			t.Fatalf("无候选必须零写入，实际 %d 次 Update（ids=%v）", fake.updateCalls, fake.updatedIDs)
		}
	})
}

// 有候选 ⇒ 按 _id 逐个更新：每个候选一次 Update，filter 只含 _id，$set healed_at +
// $setOnInsert 补全台账键；predicate 三条件齐备且排除已修复行；边界（to == watermark）
// 相触即算越过；healed_at 字段缺失视为未修复。
func TestReconcileHealedParts_UpdatesEachCandidateByID(t *testing.T) {
	fake := newReconcileFake()
	first := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	fake.seedHole(100, 110, FailureClassState, nil)       // to=110 <= 150 ⇒ 修
	fake.seedHole(120, 150, FailureClassData, nil)        // to=150 == 150 ⇒ 修（相触即算越过）
	fake.seedHoleNoHealedField(50, 60, FailureClassState) // 字段缺失 = 未修复，to=60 <= 150 ⇒ 修
	fake.seedHole(200, 210, FailureClassState, nil)       // to=210 > 150 ⇒ 不动
	fake.seedHole(10, 20, FailureClassState, &first)      // 已修复 ⇒ 非候选、不覆盖首次时间
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 150, []string{FailureClassState, FailureClassData}); err != nil {
		t.Fatalf("对账不应报错: %v", err)
	}

	want := []string{
		model.SkippedPartID(50, 60),
		model.SkippedPartID(100, 110),
		model.SkippedPartID(120, 150),
	}
	if fake.updateCalls != len(want) {
		t.Fatalf("每个候选应各发一次 Update（共 %d），实际 %d（ids=%v）", len(want), fake.updateCalls, fake.updatedIDs)
	}
	for _, id := range want {
		if !stringInList(id, fake.updatedIDs) {
			t.Fatalf("候选 %s 应被按 _id 更新，实际更新了 %v", id, fake.updatedIDs)
		}
	}
	if stringInList(model.SkippedPartID(200, 210), fake.updatedIDs) {
		t.Fatal("水位未越过的洞不得被更新")
	}
	if stringInList(model.SkippedPartID(10, 20), fake.updatedIDs) {
		t.Fatal("已修复的洞不得被更新（不在候选里）")
	}

	// predicate 形状（与旧 updateMany 完全相同的三条件）。
	if v, ok := fake.findFilter["healed_at"]; !ok || v != nil {
		t.Fatalf("predicate.healed_at 必须是 null 判据，实际 %v", fake.findFilter["healed_at"])
	}
	if tc, ok := fake.findFilter["to_epoch"].(bson.M); !ok || tc["$lte"] != abi.ChainEpoch(150) {
		t.Fatalf("predicate.to_epoch 必须是 {$lte:150}，实际 %v", fake.findFilter["to_epoch"])
	}
	ec, ok := fake.findFilter["error_class"].(bson.M)
	if !ok {
		t.Fatalf("predicate.error_class 必须是 $in，实际 %v", fake.findFilter["error_class"])
	}
	if list, _ := ec["$in"].([]string); len(list) != 2 {
		t.Fatalf("predicate.error_class.$in 应含 state+data 两类，实际 %v", ec["$in"])
	}
	if fake.findLimit != reconcileCandidateLimit {
		t.Fatalf("候选查询应带上限 %d，实际 %d", reconcileCandidateLimit, fake.findLimit)
	}

	// 更新文档形状：$set healed_at 非空 + $setOnInsert 补全台账键（防竞态插垃圾行）。
	set, ok := fake.lastUpdateDocs["$set"].(bson.M)
	if !ok || set["healed_at"] == nil {
		t.Fatalf("Update 必须 $set healed_at，实际 %v", fake.lastUpdateDocs)
	}
	soi, ok := fake.lastUpdateDocs["$setOnInsert"].(bson.M)
	if !ok {
		t.Fatalf("Update 必须带 $setOnInsert（双保险：竞态插行也要是完整台账行），实际 %v", fake.lastUpdateDocs)
	}
	for _, k := range []string{"from_epoch", "to_epoch", "error_class"} {
		if _, ok := soi[k]; !ok {
			t.Fatalf("$setOnInsert 必须补全 %q，实际 %v", k, soi)
		}
	}

	// 落账效果。
	if v := healedAtOf(fake, 100, 110); v == nil {
		t.Fatal("越过的洞应被标 healed_at")
	}
	if v := healedAtOf(fake, 120, 150); v == nil {
		t.Fatal("边界相触（to == watermark）的洞应被标 healed_at")
	}
	if v := healedAtOf(fake, 50, 60); v == nil {
		t.Fatal("healed_at 字段缺失的洞应被标 healed_at")
	}
	if v := healedAtOf(fake, 200, 210); v != nil {
		t.Fatalf("水位未越过的洞必须保持未修复，实际 healed_at=%v", v)
	}
	if got, ok := healedAtOf(fake, 10, 20).(*time.Time); !ok || !got.Equal(first) {
		t.Fatalf("已修复的洞必须保留首次 heal 时间 %v，实际 %v", first, healedAtOf(fake, 10, 20))
	}
}

// 水位未越过任何洞末端 ⇒ 候选为空 ⇒ 零 Update，且不动台账。
func TestReconcileHealedParts_WatermarkBeforeHoleIssuesZeroUpdates(t *testing.T) {
	fake := newReconcileFake()
	fake.seedHole(100, 110, FailureClassState, nil)
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 99, []string{FailureClassState}); err != nil {
		t.Fatalf("对账不应报错: %v", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("水位 99 未越过末端 110 ⇒ 零 Update，实际 %d", fake.updateCalls)
	}
	if v := healedAtOf(fake, 100, 110); v != nil {
		t.Fatalf("不应改动该行，实际 healed_at=%v", v)
	}
}

// 已有 healed_at ⇒ 不在候选里、不覆盖（保留首次 heal 时间）。
func TestReconcileHealedParts_DoesNotOverwriteExistingHealTime(t *testing.T) {
	fake := newReconcileFake()
	first := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	fake.seedHole(10, 20, FailureClassState, &first)
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 1000, []string{FailureClassState}); err != nil {
		t.Fatalf("对账不应报错: %v", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("已修复的行不在候选里，不应更新，实际 %d 次（ids=%v）", fake.updateCalls, fake.updatedIDs)
	}
	got, ok := healedAtOf(fake, 10, 20).(*time.Time)
	if !ok || !got.Equal(first) {
		t.Fatalf("必须保留首次 heal 时间 %v，实际 %v", first, healedAtOf(fake, 10, 20))
	}
}

// watermark <= 0 ⇒ no-op：既不查候选、也不写库。
func TestReconcileHealedParts_NonPositiveWatermarkIsNoOp(t *testing.T) {
	for _, wm := range []abi.ChainEpoch{0, -1, -1000} {
		fake := newReconcileFake()
		fake.seedHole(1, 2, FailureClassState, nil)
		swapHealFinder(t, fake.findCandidates)

		s := &Segment{db: fake}
		if err := s.ReconcileHealedParts(context.Background(), wm, []string{FailureClassState}); err != nil {
			t.Fatalf("watermark=%d 应为 no-op（nil error），实际 %v", wm, err)
		}
		if fake.updateCalls != 0 {
			t.Fatalf("watermark=%d 不应写库，实际 %d 次", wm, fake.updateCalls)
		}
		if fake.findFilter != nil {
			t.Fatalf("watermark=%d 不应发起候选查询", wm)
		}
		if v := healedAtOf(fake, 1, 2); v != nil {
			t.Fatalf("watermark=%d 不应改动台账", wm)
		}
	}
}

// 写库不可用：必须返回 error（与 recordSkippedPart 同口径，写库不可用要能被发现）。
func TestReconcileHealedParts_NilDBReturnsError(t *testing.T) {
	s := &Segment{}
	if err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState}); err == nil {
		t.Fatal("写库为 nil 时必须返回 error")
	}
}

// classes 过滤生效：只对本实例负责的类别背书。
func TestReconcileHealedParts_ClassesFilterApplies(t *testing.T) {
	fake := newReconcileFake()
	fake.seedHole(10, 20, FailureClassState, nil)
	fake.seedHole(30, 40, FailureClassData, nil)
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState}); err != nil {
		t.Fatalf("对账不应报错: %v", err)
	}
	if v := healedAtOf(fake, 10, 20); v == nil {
		t.Fatal("state 洞应被修复")
	}
	if v := healedAtOf(fake, 30, 40); v != nil {
		t.Fatalf("classes 只含 state 时 data 洞不得被修复，实际 healed_at=%v", v)
	}
	if fake.updateCalls != 1 || fake.updatedIDs[0] != model.SkippedPartID(10, 20) {
		t.Fatalf("应只更新 state 那一行，实际 %d 次 %v", fake.updateCalls, fake.updatedIDs)
	}
}

// 空 classes ⇒ no-op（不把「谁都没抽过的洞」错标为已修复）：既不查候选、也不写库。
func TestReconcileHealedParts_EmptyClassesIsNoOp(t *testing.T) {
	fake := newReconcileFake()
	fake.seedHole(10, 20, FailureClassState, nil)
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 100, nil); err != nil {
		t.Fatalf("空 classes 应为 no-op: %v", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("空 classes 不应写库，实际 %d 次", fake.updateCalls)
	}
	if fake.findFilter != nil {
		t.Fatal("空 classes 不应发起候选查询")
	}
}

// 候选查询报错 ⇒ 返回 error（失败关闭：写库不可用要能被发现）。走默认 findHealCandidates。
func TestReconcileHealedParts_FindErrorReturnsError(t *testing.T) {
	fake := newReconcileFake()
	fake.findErr = errors.New("mongo down")

	s := &Segment{db: fake}
	err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState})
	if err == nil {
		t.Fatal("候选查询报错必须返回 error")
	}
	if !strings.Contains(err.Error(), "list candidates") {
		t.Fatalf("错误正文应标明失败在候选查询阶段，实际 %v", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("查询失败不得写库，实际 %d 次 Update", fake.updateCalls)
	}
}

// 候选查询返回 nil 游标 ⇒ 返回 error（MultiDB 无可写库时返回 (nil,nil)，不是「没有候选」）。
func TestReconcileHealedParts_NilCursorReturnsError(t *testing.T) {
	fake := newReconcileFake() // 默认 Find 返回 (nil, nil)

	s := &Segment{db: fake}
	err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState})
	if err == nil {
		t.Fatal("nil 游标必须返回 error（不得当成零候选而静默放行）")
	}
	if !strings.Contains(err.Error(), "nil cursor") {
		t.Fatalf("错误正文应点明 nil cursor，实际 %v", err)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("nil 游标不得写库，实际 %d 次 Update", fake.updateCalls)
	}
}

// 底层写失败：必须把错误透出（调用方据此只记日志，不影响批次）。
func TestReconcileHealedParts_UpdateErrorPropagates(t *testing.T) {
	fake := newReconcileFake()
	fake.seedHole(10, 20, FailureClassState, nil) // 造一个候选，才会真的走到 Update
	fake.updateErr = errors.New("mongo down")
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState}); err == nil {
		t.Fatal("底层写失败必须透出 error")
	}
}

// 日志正文与触发条件（主机四判据检测器按 grep "healed" 判 heal 闭环）：
//   - 真的销项（有候选）⇒ 恰好一条 Info，正文含 "healed"；
//   - 无候选的常规轮次 ⇒ 一条含 "healed" 的日志都不许打（否则检测器天天假阳性）。
func TestReconcileHealedParts_HealedLogOnlyOnActualReconcile(t *testing.T) {
	t.Run("有候选⇒恰好一条含healed的Info", func(t *testing.T) {
		logs := captureSegmentLogs(t)
		fake := newReconcileFake()
		fake.seedHole(10, 20, FailureClassState, nil)
		swapHealFinder(t, fake.findCandidates)

		s := &Segment{db: fake}
		if err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState}); err != nil {
			t.Fatalf("对账不应报错: %v", err)
		}
		n := 0
		for _, e := range logs.All() {
			if e.Level == zap.InfoLevel && strings.Contains(e.Message, "healed") {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("销项后应恰好一条含 \"healed\" 的 Info，实际 %d（entries=%v）", n, logs.All())
		}
	})

	t.Run("无候选⇒无含healed的日志", func(t *testing.T) {
		logs := captureSegmentLogs(t)
		fake := newReconcileFake() // 空台账 ⇒ 无候选
		swapHealFinder(t, fake.findCandidates)

		s := &Segment{db: fake}
		if err := s.ReconcileHealedParts(context.Background(), 100, []string{FailureClassState}); err != nil {
			t.Fatalf("对账不应报错: %v", err)
		}
		for _, e := range logs.All() {
			if strings.Contains(e.Message, "healed") {
				t.Fatalf("无候选轮次不得打含 \"healed\" 的日志（检测器会假阳性），实际 %v", e)
			}
		}
	})
}

// 候选数达上限 ⇒ 记一条 warn（提示可能还有更多未销项，下一轮继续）。
func TestReconcileHealedParts_CandidateLimitWarns(t *testing.T) {
	logs := captureSegmentLogs(t)
	fake := newReconcileFake()
	for i := 0; i < reconcileCandidateLimit; i++ {
		from := abi.ChainEpoch(i*10 + 1)
		fake.seedHole(from, from+5, FailureClassState, nil)
	}
	swapHealFinder(t, fake.findCandidates)

	s := &Segment{db: fake}
	if err := s.ReconcileHealedParts(context.Background(), abi.ChainEpoch(1<<40), []string{FailureClassState}); err != nil {
		t.Fatalf("对账不应报错: %v", err)
	}
	if fake.updateCalls != reconcileCandidateLimit {
		t.Fatalf("达上限时应处理满 %d 个候选，实际 %d", reconcileCandidateLimit, fake.updateCalls)
	}
	warned := false
	for _, e := range logs.All() {
		if e.Level == zap.WarnLevel && strings.Contains(e.Message, "candidate limit") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("候选达上限应记 warn，实际 entries=%v", logs.All())
	}
}

// classes 选择：主 bell 背书 state+data，statebell 只背书 state。
func TestHealClassesFor(t *testing.T) {
	main := healClassesFor(false)
	if len(main) != 2 || main[0] != FailureClassState || main[1] != FailureClassData {
		t.Fatalf("主 bell 应为 [state,data]，实际 %v", main)
	}
	stateOnly := healClassesFor(true)
	if len(stateOnly) != 1 || stateOnly[0] != FailureClassState {
		t.Fatalf("statebell 应为 [state]，实际 %v", stateOnly)
	}
}
