package segment

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

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
