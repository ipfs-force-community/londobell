package segment

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

// recordSkippedPart 把一个 part 失败登记进台账（集合 SkippedPart，写入 bell 的写库 s.db，
// 与 FinalHeight 同库）。
//
// 去重靠 _id = "<from>-<to>"：一条「_id 过滤 + upsert」就把「不存在则插入、存在则只更新
// last_seen / attempts」做完。底层 DocumentDB.Update 是带 Upsert=true 的 UpdateMany
// （见 lib/mgoutil/docdb.go 的 NewMgoDocDB：updateOpt = options.Update().SetUpsert(true)），
// 因此同一区间重复命中不会插新行；$setOnInsert 保证首次插入才写 first_seen / 区间键。
//
// 铁律（照抄上游）：台账写不进去就不允许跳过。返回 error 时调用方必须把这个 part 当作
// 「不可登记」处理，回到「立即整批作废」的现状行为（否则就成了静默跳洞）。
func (s *Segment) recordSkippedPart(ctx context.Context, l *zap.SugaredLogger, pf PartFailure) error {
	if s.db == nil {
		// 写库不可用 = 台账写不进去 = 不允许跳过。
		return fmt.Errorf("record skipped part: write db is nil")
	}

	now := time.Now()
	id := model.SkippedPartID(pf.FromEpoch, pf.ToEpoch)

	update := bson.M{
		// 每次命中都刷新：最近一次错误形态与时间。
		"$set": bson.M{
			"error_class": pf.Class,
			"err_digest":  pf.ErrDigest,
			"cid":         pf.CID,
			"last_seen":   now,
		},
		// 只在插入时写：首见时间与区间键。healed_at 交给后续的 heal 闭环，本段不碰
		// （放在 $setOnInsert 里就永远不会覆盖将来被 heal 置上的值）。
		"$setOnInsert": bson.M{
			"from_epoch": pf.FromEpoch,
			"to_epoch":   pf.ToEpoch,
			"first_seen": now,
			"healed_at":  nil,
		},
		// attempts 累加：同一区间每命中一次 +1。
		"$inc": bson.M{"attempts": 1},
	}

	if _, err := s.db.Update(ctx, model.SkippedPartCollection(), bson.M{"_id": id}, update); err != nil {
		return fmt.Errorf("update skipped part ledger %s: %w", id, err)
	}

	l.Warnw("skipped part ledger updated",
		"id", id, "class", pf.Class,
		"from", int64(pf.FromEpoch), "to", int64(pf.ToEpoch), "cid", pf.CID)

	return nil
}

// HoleSpan 是台账里一段「已登记且尚未修复」的洞的高度区间——跨洞防护判据的输入。
// 只带判定与报错所需的字段，不含 err_digest 等正文。
type HoleSpan struct {
	FromEpoch abi.ChainEpoch
	ToEpoch   abi.ChainEpoch
	Class     string
}

// RegisteredHolesInRange 查询台账（集合 SkippedPart）里与闭区间 [lo,hi] 重叠、
// 且尚未修复（healed_at 为空或字段缺失）的洞。这是运维手动聚合入口（RaCailum.Aggregate）
// 的跨洞防护所依赖的唯一台账查询（docs/cn/batch_failure_semantics.md §8.11 的遗留项）。
//
// 为何读写库 s.db 而不是读库 s.rdb：洞刚登记就要拦下一次聚合，读库（若配置了副本）存在
// 复制延迟，可能读到「还没有这个洞」而放行——跨洞防护读到的必须是台账的最新真值。
//
// 重叠判据（闭区间）：洞 [from,to] 与 [lo,hi] 相交 ⇔ from ≤ hi 且 to ≥ lo。
// 边界相触即算相交（洞恰好落在 lo 或 hi 上也必须拦），故用 ≤ / ≥ 而非 < / >。
//
// 失败关闭：写库不可用、查询报错、或拿不到游标，一律返回 error（而不是空结果），
// 让调用方在「不知道有没有洞」时拒绝聚合，而不是赌数据正确性。
func (s *Segment) RegisteredHolesInRange(ctx context.Context, lo, hi abi.ChainEpoch) ([]HoleSpan, error) {
	if s.db == nil {
		return nil, fmt.Errorf("query skipped part ledger: db is nil")
	}

	filter := bson.M{
		"from_epoch": bson.M{"$lte": hi},
		"to_epoch":   bson.M{"$gte": lo},
		// healed_at 为 null 或字段缺失 ⇒ 尚未修复（recordSkippedPart 首次插入写的就是 nil）。
		// 已被 heal 置上时间的洞不再拦。
		"healed_at": nil,
	}
	opts := options.Find().SetSort(bson.D{{Key: "from_epoch", Value: 1}})

	cursor, err := s.db.Find(ctx, model.SkippedPartCollection(), filter, opts)
	if err != nil {
		return nil, fmt.Errorf("query skipped part ledger [%d-%d]: %w", lo, hi, err)
	}
	if cursor == nil {
		// MultiDB 在没有任何可写库时会返回 (nil, nil)——这不是「没有洞」，是「查不到」。
		return nil, fmt.Errorf("query skipped part ledger [%d-%d]: nil cursor", lo, hi)
	}

	var rows []model.SkippedPart
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode skipped part ledger [%d-%d]: %w", lo, hi, err)
	}

	holes := make([]HoleSpan, 0, len(rows))
	for _, r := range rows {
		holes = append(holes, HoleSpan{FromEpoch: r.FromEpoch, ToEpoch: r.ToEpoch, Class: r.ErrorClass})
	}
	return holes, nil
}

// firstHoleOverlappingRange 返回 holes 中第一个与 [lo,hi] 重叠的洞（holes 已按 from_epoch 升序）。
// 纯函数、无副作用：把「区间重叠」这条判据从「查台账」这一 IO 里分离出来，让四种区间关系
// （无洞 / 洞在中间 / 洞贴 lo 或 hi 边界 / 完全不重叠）都能脱离 mongo 表驱动验证。
func firstHoleOverlappingRange(holes []HoleSpan, lo, hi abi.ChainEpoch) (HoleSpan, bool) {
	for _, h := range holes {
		if h.FromEpoch <= hi && h.ToEpoch >= lo {
			return h, true
		}
	}
	return HoleSpan{}, false
}

// CheckNoCrossHoleAggregation 是运维手动聚合入口（RaCailum.Aggregate）的「跨洞防护」判据：
// 先查台账，再判 [lo,hi] 是否覆盖到已登记且未修复的洞。
//
//   - 台账查询失败 ⇒ **失败关闭**：返回 error（含原始错误）。语义理由：查不到台账 = 不知道
//     有没有洞，此时放行等于赌数据正确性；宁可拒绝聚合，让运维先修台账/重试。
//   - 查到洞 ⇒ 返回 error，正文含洞的高度区间（运维据此知道该先 heal 哪一段）。
//     **不自动截断到前缀**：手动聚合是运维要一个完整区间，静默只给一半结果同样会被误读为
//     「聚合成功」，与跨洞发布同属要消灭的错误形态。
//   - 无洞 ⇒ 返回 nil（放行）。
//
// query 以函数形式注入（口径同 prefixCommitDeps 的回调注入）：把「查台账」这一 IO 从判据里
// 剥离，单测可注入可控的洞集合，不必连 mongo。生产侧传入 (*Segment).RegisteredHolesInRange。
func CheckNoCrossHoleAggregation(ctx context.Context, query func(context.Context, abi.ChainEpoch, abi.ChainEpoch) ([]HoleSpan, error), lo, hi abi.ChainEpoch) error {
	holes, err := query(ctx, lo, hi)
	if err != nil {
		return fmt.Errorf("cross-hole aggregation refused: skipped part ledger query failed: %w", err)
	}

	if h, ok := firstHoleOverlappingRange(holes, lo, hi); ok {
		return fmt.Errorf("cross-hole aggregation refused: range [%d,%d] covers registered unhealed hole [%d,%d] (class=%s); heal it before aggregating",
			lo, hi, h.FromEpoch, h.ToEpoch, h.Class)
	}
	return nil
}

// healClassesFor 返回本实例有权为其「背书已修复」的错误类别，即它实际抽取过的类别。
//
//   - 主 bell（onlyExtractState == false）做全量抽取 ⇒ 对 state 与 data 两类洞都背书；
//   - statebell（onlyExtractState == true）只抽状态、只前移状态水位 ⇒ 只对 state 类背书。
//
// 为什么必须区分：heal 判据用的是「本实例已发布的水位越过了洞末端」，而水位只证明
// 「本实例抽取过的数据被重抽且落库成功」。statebell 的水位前移**不代表** data 数据被抽过，
// 拿它去证明一个 data 洞已修复是越权证明。
func healClassesFor(onlyExtractState bool) []string {
	if onlyExtractState {
		return []string{FailureClassState}
	}
	return []string{FailureClassState, FailureClassData}
}

// reconcileCandidateLimit 是单轮对账最多处理的候选行数上限。取保守值：一轮抓不满时下一轮
// （水位不变）会继续处理剩余候选，不会有洞被漏掉；上限只为避免在异常大的台账上单轮耗尽内存。
const reconcileCandidateLimit = 1000

// healCandidateFinder 是「拉取本轮对账候选」的注入点类型：查写库、解到 []model.SkippedPart。
type healCandidateFinder func(ctx context.Context, db common.DocumentDB, filter bson.M, limit int64) ([]model.SkippedPart, error)

// findHealCandidates 是 healCandidateFinder 的默认实现：直接查写库。
//
// 抽成函数变量是为了单测能注入可控候选集与错误（口径同 CheckNoCrossHoleAggregation 的 query
// 回调注入）——mongo-driver v1.5.0 无 NewCursorFromDocuments，单测无法凭空造出可被 cursor.All
// 解码的 *mongo.Cursor，只能把这步 Find+All 整体做成注入点。
var findHealCandidates healCandidateFinder = func(ctx context.Context, db common.DocumentDB, filter bson.M, limit int64) ([]model.SkippedPart, error) {
	cursor, err := db.Find(ctx, model.SkippedPartCollection(), filter, options.Find().SetLimit(limit))
	if err != nil {
		return nil, err
	}
	if cursor == nil {
		// MultiDB 在没有任何可写库时会返回 (nil, nil)——这不是「没有候选」，是「查不到」。
		return nil, fmt.Errorf("find skipped part candidates: nil cursor")
	}
	var rows []model.SkippedPart
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode skipped part candidates: %w", err)
	}
	return rows, nil
}

// ReconcileHealedParts 把台账里「已被本实例已发布的最终水位越过末端」的洞标记为已修复
// （$set healed_at = now）。这是 model.SkippedPart.HealedAt 的唯一写入路径——补上台账的
// heal 闭环（见 docs/cn/batch_failure_semantics.md §8.4 的 healed_at 字段与 §8.9 验收第 4 条）。
//
// 判据（单一、可证）：本实例**已发布的最终水位** watermark 越过洞的末端（to_epoch ≤ watermark）。
// 水位只在本轮真的抽取**并落库成功**之后才前移（见 prefix_commit.go 的 commitPrefix：
// 先 Aggregate、再 SaveFinalHeight / SaveStateFinalHeight），所以「to_epoch ≤ watermark」
// 本身就等价于「这段数据已重抽且正确」——不需要（也无法）另读数据本身来复核。
//
// classes 是本实例负责（= 实际抽取过）的类别，只有这些类别的洞才由本实例背书；取值见 healClassesFor。
//
// 只判「尚未修复」的行（healed_at 为 null 或字段缺失）：已在更早轮次被标记的洞保留其**首次**
// heal 时间，不被本轮覆盖（「何时缓和」比「最后一次复核缓和」更有信息量）。
//
// 实现形状：**先按 predicate 查候选（只读），再对每个候选按 _id 逐个 $set healed_at**。
// 绝不用「无等值条件的一条 updateMany」：底层 DocumentDB.Update 恒带 Upsert=true
// （lib/mgoutil/docdb.go 的 NewMgoDocDB：updateOpt = options.Update().SetUpsert(true)），
// 而无等值条件的 filter 一旦在某一轮没有任何匹配行，就会 upsert 出一行垃圾台账（只有
// _id + healed_at，缺 from/to/class），每轮 +1 行、约 1000 行/天。稳态（台账里没有未修复的洞）
// 下「无匹配」正是每轮的形状 ⇒ 必须先查候选，候选为空就直接返回、零写入。
//
// 边界（与 recordSkippedPart / RegisteredHolesInRange 同口径）：
//   - watermark ≤ 0 ⇒ 直接 return nil（no-op）：「非正水位」不是「越过了洞」，不能误伤；
//   - db == nil ⇒ 返回 error：写库不可用必须能被发现（调用方据此只记日志、不影响批次）；
//   - classes 为空 ⇒ return nil：没有可背书的类别，不把「谁都没抽过的洞」错标为已修复；
//   - 候选查询报错 / 拿不到游标 ⇒ 返回 error（与 RegisteredHolesInRange 同口径：写库不可用
//     要能被发现）；无候选 ⇒ 直接 return nil（零写入）。
func (s *Segment) ReconcileHealedParts(ctx context.Context, watermark abi.ChainEpoch, classes []string) error {
	if watermark <= 0 {
		// 非正水位（0 / 负数）绝不代表「越过洞末端」，直接不动台账。
		return nil
	}
	if s.db == nil {
		return fmt.Errorf("reconcile healed parts: write db is nil")
	}
	if len(classes) == 0 {
		return nil
	}

	// 与旧「一条 updateMany」完全相同的三条件 predicate：未修复 + 水位越过洞末端 + 类别在册。
	predicate := bson.M{
		// healed_at 为 null 或字段缺失 ⇒ 尚未修复（recordSkippedPart 首次插入写的就是 nil）。
		// 已被 heal 的行不被命中 ⇒ 不覆盖首次 heal 时间。
		"healed_at": nil,
		// 已发布水位越过洞末端 ⇒ 该段数据已重抽且落库成功。
		"to_epoch": bson.M{"$lte": watermark},
		// 只对本实例负责（实际抽取过）的类别背书。
		"error_class": bson.M{"$in": classes},
	}

	candidates, err := findHealCandidates(ctx, s.db, predicate, reconcileCandidateLimit)
	if err != nil {
		return fmt.Errorf("reconcile healed parts: list candidates (watermark %d, classes %v): %w", watermark, classes, err)
	}
	if len(candidates) == 0 {
		// ★ 关键：无候选 ⇒ 零写入。这条 return 让稳态（台账里没有未修复的洞）不再产生任何
		// Update 调用，从而根治「无匹配 filter 的 upsert 每轮插一行垃圾台账」的缺陷。
		return nil
	}

	now := time.Now()
	for _, c := range candidates {
		update := bson.M{
			"$set": bson.M{"healed_at": now},
			// 双保险：万一在「查候选」与「逐 _id 更新」之间该行被删/被改（竞态）而触发 upsert，
			// 插进去的也必须是一行**完整正确**的台账行，而不是只有 _id + healed_at 的垃圾行。
			"$setOnInsert": bson.M{
				"from_epoch":  c.FromEpoch,
				"to_epoch":    c.ToEpoch,
				"error_class": c.ErrorClass,
			},
		}
		if _, err := s.db.Update(ctx, model.SkippedPartCollection(), bson.M{"_id": c.ID}, update); err != nil {
			return fmt.Errorf("reconcile healed part %s: %w", c.ID, err)
		}
	}

	// 真的销项了（healedCount > 0）才打这条 Info，且**正文含 "healed"**：主机上的四判据检测器
	// 就是 grep bell 日志里的 "healed" 来判「验收第 4 条 heal 闭环」。无候选的常规轮次在上面
	// 已经 return，绝不打 —— 否则检测器会天天报假阳性。
	log.Infow("skipped part ledger reconciled: holes healed",
		"watermark", int64(watermark), "classes", classes, "healed", len(candidates))

	if len(candidates) >= reconcileCandidateLimit {
		log.Warnw("reconcile healed parts reached candidate limit; more may remain next round",
			"watermark", int64(watermark), "classes", classes, "limit", reconcileCandidateLimit)
	}

	return nil
}
