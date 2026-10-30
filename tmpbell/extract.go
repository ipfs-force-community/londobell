package tmpbell

import (
	"context"
	"time"

	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/metrics"

	"github.com/filecoin-project/go-state-types/abi"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/types"
)

func (t *TmpBell) ExtractIncomingHead(ctx context.Context, tempDBCapacity uint) {
	for {
		start := time.Now()
		head, err := t.Full.GetAppropriateAPI().ChainHead(ctx)
		if err != nil {
			log.Errorf("get chain head: %s", err)
			stats.Record(ctx, metrics.GetTipSetError.M(1))
			time.Sleep(5 * time.Second)
			continue
		}
		stats.Record(ctx, metrics.GetTipSetError.M(0))

		tmpFinalTipSet, err := t.activeSeg.GetTmpFinalTipSet(ctx)
		if err != nil {
			log.Errorf("get latestExtractTipset for tmp db failed: %s", err)
			stats.Record(ctx, metrics.GetTipSetError.M(1))
			time.Sleep(5 * time.Second)
			continue
		}
		stats.Record(ctx, metrics.GetTipSetError.M(0))

		nextExtractHeight := tmpFinalTipSet.Height() + abi.ChainEpoch(continuousNullTipSet) + 1
		if nextExtractHeight > head.Height() {
			// extract too quickly or lotus has been thinned, wait
			time.Sleep(5 * time.Second)
			continue
		}

		log.Infof("request next tipset at height %v", nextExtractHeight)
		nextExtractTipSet, err := t.Full.GetAppropriateAPI().ChainGetTipSetByHeight(ctx, nextExtractHeight, types.EmptyTSK)
		if err != nil {
			log.Errorf("get next tipset: %s", err)
			stats.Record(ctx, metrics.GetTipSetError.M(1))
			time.Sleep(5 * time.Second)
			continue
		}
		stats.Record(ctx, metrics.GetTipSetError.M(0))

		exist, err := t.ForkExist(tmpFinalTipSet, nextExtractTipSet)
		if err != nil && err != errNullTipSet {
			log.Errorf("when determine if fork exists failed: %s", err)
			stats.Record(ctx, metrics.GetTipSetError.M(1))
			continue
		}
		stats.Record(ctx, metrics.GetTipSetError.M(0))

		if exist {
			parent, err := t.activeSeg.GetTipSetByTSk(ctx, nextExtractTipSet.Parents())
			if err != nil {
				log.Errorf("get fork tipset's parent failed: %s", err)
				stats.Record(ctx, metrics.GetTipSetError.M(1))
				time.Sleep(5 * time.Second)
				continue
			}
			stats.Record(ctx, metrics.GetTipSetError.M(0))

			log.Infof("fork exists, latestExtractTipset: %v, nextTs: %v, parent: %v", tmpFinalTipSet, nextExtractTipSet, parent)
			err = t.HandleFork(ctx, tmpFinalTipSet, nextExtractTipSet, tempDBCapacity)
			if err != nil {
				log.Errorf("handle fork failed: %s", err)
				if err == errNoAncestor {
					// todo: 增加tempDBCapacity（人工or自动）
					log.Error("need to adjust fork height range setting")
					stats.Record(ctx, metrics.TempDBCapacityError.M(1))
					return
				}
				stats.Record(ctx, metrics.GetTipSetError.M(1))
				continue
			}
			stats.Record(ctx, metrics.GetTipSetError.M(0))

			// 分叉后**不要**清 actor set 缓存：缓存里有 772 万条（主网实测
			// count=3855373 + init-state=3871211）actor code 映射，清一次就要全量重建，
			// 实测 4.5~6.5 分钟，这段时间实时水位（tmp 库最新 tipset）不前进。
			// 主网分叉极密（2026-10-30 实测 tmp.log 11 小时内 18 次，约每 20~40 分钟一次），
			// 每次清缓存 ⇒ 每次 5 分钟停更；一旦与「预算/窗口」常规重建撞在同一高度，
			// 就是 12 分钟停更（当天 05:16 / 08:18 / 12:18 各一次，delay 峰值 751s 触警）。
			// 不复用的正确性由两处兜底，故这里可以安全地保留缓存：
			//  1) acquireActorSet 的有界失效：网络版本变化 / 30 轮复用预算 / epoch 窗口，
			//     条件满足时仍会重建，分叉造成的陈旧最多持续到下一次有界失效；
			//  2) LookupMethodInfo 的逐消息兜底：缓存查不到的 actor 会走
			//     stm.LoadActor(ctx, call.To, ts) 从该 tipset 状态读取并回填缓存
			//     （分叉新增的 actor 正好走这条路）。
			log.Infof("handle fork successfully")
			continue
		}

		if err == errNullTipSet {
			log.Warnw("null tipSet exists", "tmpFinalHeight", tmpFinalTipSet.Height(), "nextExtractHeight", nextExtractHeight)
			continue
		}

		continuousNullTipSet = 0

		err = t.PrepareExtractToTemporaryDB(ctx, nextExtractTipSet, tempDBCapacity)
		if err != nil {
			log.Warn("error occurs when extract to temporary db", err)
			stats.Record(ctx, metrics.ExtractError.M(1))
			continue
		}

		stats.Record(ctx, metrics.ExtractError.M(0))
		log.Infof("extract tipset %v to temporary db spent: %v", nextExtractTipSet.Height(), time.Now().Sub(start).String())
	}
}

// UpdateTemporaryBoundary updates finalHeight to boundary.hi
func (t *TmpBell) UpdateTemporaryBoundary(ctx context.Context, finalHeight abi.ChainEpoch) error {
	finalTipSet, err := t.Full.GetAppropriateAPI().ChainGetTipSetByHeight(ctx, finalHeight, types.EmptyTSK)
	if err != nil {
		log.Errorf("get tipset at curFormalDBHeight failed: %v", err)
		return err
	}

	hi, err := LoadLinkedTipSet(ctx, finalTipSet.Key(), t.Full)
	if err != nil {
		return err
	}

	err = t.activeSeg.SetBoundary(ctx, hi, nil)
	if err != nil {
		return err
	}

	return nil
}

func (t *TmpBell) ClearTmpDB(ctx context.Context) error {
	tlog := log.With("call", "ClearTmpDB")
	err := t.activeSeg.DeleteItemsByEpoch(ctx, tlog, -1, true, false)
	if err != nil {
		return err
	}

	return nil
}

func (t *TmpBell) ForkExist(latestExtractTipset, nextTs *types.TipSet) (bool, error) {
	if types.CidArrsEqual(latestExtractTipset.Cids(), nextTs.Cids()) {
		continuousNullTipSet++
		log.Warnf("there is a null tipset after height %v", nextTs.Height())
		return false, errNullTipSet
	}

	return !types.CidArrsEqual(latestExtractTipset.Cids(), nextTs.Parents().Cids()), nil
}

func (t *TmpBell) HandleFork(ctx context.Context, tmpFinalTipSet, nextExtractTipSet *types.TipSet, tempDBCapacity uint) error {
	tlog := log.With("HandleFork", nextExtractTipSet.Height())

	//在临时表中寻找公共祖先
	ancestor, err := t.SearchCommonAncestor(ctx, tmpFinalTipSet, nextExtractTipSet, tempDBCapacity)
	if err != nil && err != errNoAncestor {
		return err
	}

	// clear temporary db if there id no common ancestor found, update
	if err == errNoAncestor {
		// todo: 需要调整分叉高度范围设定,直接报错返回?
		tlog.Errorf("need to adjust fork height range setting")
		return err
	}

	tlog.Warnw("fork exits in temporary db", "ancestor", ancestor.Height(), "nextExtractTipSet", nextExtractTipSet.Height(), "tmpFinalTipSet", tmpFinalTipSet.Height())

	// delete latest items in tmp db until ancestor(exclusive)   todo：需要比较此时latestExtractHeight和正式库最新高度吗？
	err = t.activeSeg.DeleteItemsByEpoch(ctx, tlog, ancestor.Height(), true, false)
	if err != nil {
		return err
	}

	// assign ancestor to boundary.hi
	err = t.UpdateTemporaryBoundary(ctx, ancestor.Height())
	if err != nil {
		return err
	}

	return nil
}

func (t *TmpBell) SearchCommonAncestor(ctx context.Context, base, external *types.TipSet, tempDBCapacity uint) (*types.TipSet, error) {
	if base == nil || external == nil {
		return nil, errZeroHeight //todo
	}

	var err error

	for forkLength := uint(0); forkLength < tempDBCapacity; forkLength++ {
		for external.Height() > base.Height() {
			if external.Height() == 0 {
				return nil, errZeroHeight
			}

			external, err = t.activeSeg.GetTipSetByTSk(ctx, external.Parents())
			if err != nil {
				log.Warnf("get external.Parent from full node")
				external, err = t.Full.GetAppropriateAPI().ChainGetTipSet(ctx, external.Parents())
				if err != nil {
					return nil, err
				}
			}
		}

		if base.Equals(external) {
			return base, nil
		}

		if base.Height() == 0 {
			return nil, errZeroHeight
		}

		//此时lotus可能已经同步？？
		base, err = t.activeSeg.GetTipSetByTSk(ctx, base.Parents())
		if err != nil {
			return nil, err
		}
	}

	return nil, errNoAncestor
}

func (t *TmpBell) PrepareExtractToTemporaryDB(ctx context.Context, ts *types.TipSet, tempDBCapacity uint) error {
	tlog := log.With("prepare to extract the height", ts.Height())

	// ensure the height span does not exceed tempDBCapacity
	err := t.DeleteEarlierItems(ctx, tlog, tempDBCapacity)
	if err != nil {
		return err
	}

	return t.activeSeg.ExtractToTemporaryDB(ctx, ts)
}

// DeleteEarlierItems deletes earlier items when count of Tipset items exceeds tempDBCapacity
func (t *TmpBell) DeleteEarlierItems(ctx context.Context, l *zap.SugaredLogger, tempDBCapacity uint) error {
	count, err := t.activeSeg.GetTipSetItemsCount(ctx, l)
	if err != nil {
		return err
	}

	if count > tempDBCapacity {
		log.Infow("delete earlier items", "count", count, "tempDBCapacity", tempDBCapacity)
		latestHeight, err := t.activeSeg.GetLatestHeightForTipSet(ctx, l)
		if err != nil {
			return err
		}

		err = t.activeSeg.DeleteItemsByEpoch(ctx, l, latestHeight-abi.ChainEpoch(tempDBCapacity), true, true)
		if err != nil {
			return err
		}
	}

	return nil
}
