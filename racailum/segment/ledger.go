package segment

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
