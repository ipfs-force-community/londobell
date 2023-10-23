package multiquery

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	"github.com/filecoin-project/go-state-types/abi"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

func GetFinalHeight(ctx context.Context, cols common.Collections) (abi.ChainEpoch, error) {
	var finalHeightRes []model.FinalHeightRes

	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetFinalHeightAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "FinalHeight" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return 0, err
			}

			err = cur.All(ctx, &finalHeightRes)
			if err != nil {
				return 0, err
			}

			if len(finalHeightRes) == 0 {
				return 0, nil
			}

			return finalHeightRes[0].Epoch, nil
		}
	}

	return 0, fmt.Errorf("no FinalHeight collection")
}

func GetStateFinalHeight(ctx context.Context, cols common.Collections) (abi.ChainEpoch, error) {
	var finalHeightRes []model.FinalHeightRes

	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetFinalHeightAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "StateFinalHeight" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return 0, err
			}

			err = cur.All(ctx, &finalHeightRes)
			if err != nil {
				return 0, err
			}

			if len(finalHeightRes) == 0 {
				return 0, nil
			}

			return finalHeightRes[0].Epoch, nil
		}
	}

	return 0, fmt.Errorf("no StateFinalHeight collection")
}

func GetIncomingBlock(ctx context.Context, pipe interface{}, cols common.Collections) ([]model.BlockHeader, error) {

	var blockHeaderRes []model.BlockHeader
	for _, col := range cols.Cols {
		if col != nil && col.Name() == "OrphanBlock" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return nil, err
			}

			err = cur.All(ctx, &blockHeaderRes)

			return blockHeaderRes, err
		}
	}

	return blockHeaderRes, fmt.Errorf("no OrphanBlock collection")
}

func GetStartEpochForDeal(ctx context.Context, cols common.Collections) (int64, error) {
	var res []struct {
		Epoch int64
	}

	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetStartEpochForDealAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "NewDealProposal" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return 0, err
			}

			err = cur.All(ctx, &res)
			if err != nil {
				return 0, err
			}

			if len(res) == 0 {
				return 0, nil
			}

			return res[0].Epoch, nil
		}
	}

	return 0, fmt.Errorf("no NewDealProposal collection")
}

func GetDealIDRange(ctx context.Context, cols common.Collections, startEpoch, endEpoch int64) (uint64, uint64, error) {
	startDealID, endDealID := uint64(0), uint64(0)
	startPipe, err := util.Parse(model.Ctx{StartEpoch: startEpoch, EndEpoch: endEpoch, Sort: 1}, string(monitor.GetDealIDRangeAggregator()))
	if err != nil {
		return 0, 0, err
	}
	endPipe, err := util.Parse(model.Ctx{StartEpoch: startEpoch, EndEpoch: endEpoch, Sort: -1}, string(monitor.GetDealIDRangeAggregator()))
	if err != nil {
		return 0, 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "NewDealProposal" {
			var startRes []struct {
				DealID uint64
			}
			var endRes []struct {
				DealID uint64
			}
			startCur, err := col.Aggregate(ctx, startPipe)
			if err != nil {
				return 0, 0, err
			}

			err = startCur.All(ctx, &startRes)
			if err != nil {
				return 0, 0, err
			}

			if len(startRes) == 0 {
				return 0, 0, nil
			}

			startDealID = startRes[0].DealID

			endCur, err := col.Aggregate(ctx, endPipe)
			if err != nil {
				return 0, 0, err
			}

			err = endCur.All(ctx, &endRes)
			if err != nil {
				return 0, 0, err
			}

			if len(endRes) == 0 {
				return 0, 0, nil
			}

			endDealID = endRes[0].DealID + 1

			return startDealID, endDealID, nil
		}
	}

	return 0, 0, fmt.Errorf("no NewDealProposal collection")
}

//func GetMinerSectorBoundary(ctx context.Context, cols common.Collections, startEpoch, endEpoch int64) (Boundrary, error) {
//	var res []Boundrary
//
//	pipe, err := util.Parse(model.Ctx{StartEpoch: startEpoch, EndEpoch: endEpoch}, string(monitor.GetMinerSectorRangeAggregator()))
//	if err != nil {
//		return Boundrary{}, err
//	}
//
//	for _, col := range cols.Cols {
//		if col != nil && col.Name() == "MinerNewSectorNumber" {
//			cur, err := col.Aggregate(ctx, pipe)
//			if err != nil {
//				return Boundrary{}, err
//			}
//
//			err = cur.All(ctx, &res)
//			if err != nil {
//				return Boundrary{}, err
//			}
//
//			if len(res) == 0 {
//				return Boundrary{}, nil
//			}
//
//			return res[0], nil
//		}
//	}
//
//	return Boundrary{}, fmt.Errorf("no MinerNewSectorNumber collection")
//}

//// todo: 递归
//func determineSegments(skip, limit,requestTotalCount int64, ptype string, countUtils []CountUtil,skipTag func(int64,int64) bool,limitTag func(int64, int64) bool) (result []CountUtil) {
//	for _, countlist := range countUtils {
//		totalCount := int64(0)
//		switch ptype {
//		case "block":
//			for _, bs := range countlist.BlockStates {
//				totalCount += bs.Count
//			}
//		case
//		default:
//		}
//
//		agg := &aggUtil{startEpoch: countlist.Start, endEpoch: countlist.End,
//			BlockStates: countlist.BlockStates,
//			BlockMethodStates :countlist.BlockMethodStates,
//			ActorStates :countlist.ActorStates,
//			ActorMethodStates :countlist.ActorMethodStates,
//			ActorTransferStates     :countlist.ActorTransferStates,
//			MinedStates              :countlist.MinedStates,
//			LargeAmountTransferStates:countlist.LargeAmountTransferStates}
//
//		skipflag := skipTag(skip, totalCount)
//		limitflag := limitTag(requestTotalCount, totalCount)
//		if skipflag && limitflag {
//			agg.skip = skip
//			agg.limit = limit
//			aggLists = append(aggLists, agg)
//			break
//		}
//		if skipflag && !limitflag {
//			agg.skip = skip
//			agg.limit = limit
//			aggLists = append(aggLists, agg)
//			skip = 0
//			limit = requestTotalCount - totalCount
//			requestTotalCount = requestTotalCount - totalCount // skip 1, limit 5  tmp 8
//			continue
//		}
//		if !skipflag {
//			skip = skip - totalCount
//			requestTotalCount = skip + limit
//			continue
//		}
//	}
//}

// MultiPagingQuery 对多库进行分页查询  todo: 磁盘state更改会影响dbStates（map）吗？
func MultiPagingQuery(ctx context.Context, indexReq, limitReq int64, ptype Ptype, countUtils []CountUtil, aggregator []byte, req model.CommonReq, tableName string) ([]bson.M, error) {
	// skip、limit
	skipTag := func(skip, count int64) bool {
		return skip < count
	}
	limitTag := func(requestTotalCount, count int64) bool {
		return requestTotalCount <= count
	}

	// todo: skip=0 & limit=0 获取全量？
	skip := int64(0)
	limit := int64(math.MaxInt64)
	requestTotalCount := int64(math.MaxInt64)

	if indexReq > 0 || limitReq > 0 {
		skip = indexReq * limitReq
		limit = limitReq
		requestTotalCount = skip + limit
	}

	segmentLists := make([]*segmentUtil, 0)

	// 锁定库
	for _, countlist := range countUtils {
		totalCount := int64(0)
		switch ptype {
		case BlockStates:
			for _, bs := range countlist.BlockStates {
				totalCount += bs.Count
			}
		case BlockMethodStates:
			for _, bms := range countlist.BlockMethodStates {
				totalCount += bms.Count
			}
		case BlockHeaderMethodStates:
			totalCount = countlist.BlockHeaderMethodStates
		case ActorStates:
			totalCount = countlist.ActorStates
		case ActorMethodStates:
			totalCount = countlist.ActorMethodStates
		case ActorTransferStates:
			totalCount = countlist.ActorTransferStates
		case ActorEventStates:
			totalCount = countlist.ActorEventStates
		case MinedStates:
			totalCount = countlist.MinedStates
		case LargeAmountTransferStates:
			totalCount = countlist.LargeAmountTransferStates
		case DealState:
			totalCount = countlist.DealState
		case DealActorStates:
			totalCount = countlist.DealActorStates
		case TipSetStates:
			totalCount = countlist.TipSetStates

		default:
			return nil, fmt.Errorf("invalid type for paging: %v", ptype)
		}

		segment := &segmentUtil{start: countlist.Start, end: countlist.End, Cols: countlist.Cols,
			BlockStates:               countlist.BlockStates,
			BlockMethodStates:         countlist.BlockMethodStates,
			ActorStates:               countlist.ActorStates,
			ActorMethodStates:         countlist.ActorMethodStates,
			ActorTransferStates:       countlist.ActorTransferStates,
			MinedStates:               countlist.MinedStates,
			LargeAmountTransferStates: countlist.LargeAmountTransferStates,
			DealState:                 countlist.DealState,
			DealActorStates:           countlist.DealActorStates,
			TipSetStates:              countlist.TipSetStates,
		}

		skipflag := skipTag(skip, totalCount)
		limitflag := limitTag(requestTotalCount, totalCount)
		if skipflag && limitflag {
			segment.skip = skip
			segment.limit = limit
			segmentLists = append(segmentLists, segment)
			break
		}
		if skipflag && !limitflag {
			segment.skip = skip
			segment.limit = limit
			segmentLists = append(segmentLists, segment)
			skip = 0
			limit = requestTotalCount - totalCount
			requestTotalCount = requestTotalCount - totalCount // skip 1, limit 5  tmp 8
			continue
		}
		if !skipflag {
			skip = skip - totalCount
			requestTotalCount = skip + limit
			continue
		}
	}

	// 锁定段
	aggLists := make([]*aggUtil, 0)
	switch ptype {
	case BlockStates:
		for _, segmentList := range segmentLists {
			sort.Slice(segmentList.BlockStates, func(i, j int) bool {
				return segmentList.BlockStates[i].StartEpoch > segmentList.BlockStates[j].StartEpoch
			})

			innerAggLists := aggListsFromSegmentState(segmentList.BlockStates, segmentList.skip, segmentList.limit, segmentList.Cols, skipTag, limitTag)
			aggLists = append(aggLists, innerAggLists...)
		}
	case BlockMethodStates:
		for _, segmentList := range segmentLists {
			sort.Slice(segmentList.BlockMethodStates, func(i, j int) bool {
				return segmentList.BlockMethodStates[i].StartEpoch > segmentList.BlockMethodStates[j].StartEpoch
			})

			innerAggLists := aggListsFromSegmentState(segmentList.BlockMethodStates, segmentList.skip, segmentList.limit, segmentList.Cols, skipTag, limitTag)
			aggLists = append(aggLists, innerAggLists...)
		}
	case BlockHeaderMethodStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.BlockHeaderMethodStates,
			})
		}
	case ActorStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.ActorStates,
			})
		}
	case ActorMethodStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.ActorMethodStates,
			})
		}
	case ActorTransferStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.ActorTransferStates,
			})
		}
	case ActorEventStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.ActorEventStates,
			})
		}
	case MinedStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.MinedStates,
			})
		}
	case LargeAmountTransferStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.LargeAmountTransferStates,
			})
		}
	case DealState:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.DealState,
			})
		}
	case DealActorStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.DealActorStates,
			})
		}

	case TipSetStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.TipSetStates,
			})
		}
	default:
		return nil, fmt.Errorf("invalid type of paging: %v", ptype)
	}

	// concurrent agg
	var (
		res    = make([][]bson.M, len(aggLists))
		result = make([]bson.M, 0)
		ewg    multierror.Group
	)

	lim := limiter.New(16)

	for i := range aggLists {
		i := i
		aggList := aggLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				return nil
			}

			defer func() {
				lim.Release(ctx)
			}()

			var aggRes []bson.M

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.start, EndEpoch: aggList.end, Start: aggList.start, End: aggList.end, Skip: aggList.skip, Limit: aggList.limit, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, ID: req.ID, Sort: req.Sort, To: req.To, Addrs: req.Addrs, Addr: req.Addr}, string(aggregator)) // todo: methodName
			if err != nil {
				return err
			}

			for _, col := range aggList.cols.Cols {
				if col != nil && col.Name() == tableName {
					cur, err := col.Aggregate(ctx, pipe)
					if err != nil {
						return err
					}

					err = cur.All(ctx, &aggRes)
					if err != nil {
						return err
					}

					res[i] = aggRes

					return nil
				}
			}

			return fmt.Errorf("no collection: %v", tableName)
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		result = append(result, res[i]...)
	}

	return result, nil
}

func aggListsFromSegmentState(segmentStates []smodel.SegmentState, skip, limit int64, cols common.Collections, skipTag func(int64, int64) bool, limitTag func(int64, int64) bool) []*aggUtil {
	requestTotalCount := skip + limit
	aggLists := make([]*aggUtil, 0)
	for _, ss := range segmentStates {
		count := ss.Count

		agg := &aggUtil{start: int64(ss.StartEpoch), end: int64(ss.EndEpoch), cols: cols}
		skipflag := skipTag(skip, count)
		limitflag := limitTag(requestTotalCount, count)
		if skipflag && limitflag {
			agg.skip = skip
			agg.limit = limit
			aggLists = append(aggLists, agg)
			break
		}
		if skipflag && !limitflag {
			agg.skip = skip
			agg.limit = limit
			aggLists = append(aggLists, agg)
			skip = 0
			limit = requestTotalCount - count
			requestTotalCount = requestTotalCount - count
			continue
		}
		if !skipflag {
			skip = skip - count
			requestTotalCount = skip + limit
			continue
		}
	}

	return aggLists
}

// MultiRangeQuery 根据epoch范围定位到某些库查询
func MultiRangeQuery(ctx context.Context, startEpoch, endEpoch int64, countUtils []CountUtil, aggregator []byte, req model.CommonReq, tableName string) ([]bson.M, error) {
	start := startEpoch
	end := endEpoch
	aggLists := make([]*aggUtil, 0)
	for i := range countUtils {
		i := i
		countList := countUtils[i]
		if start >= countList.Start {
			// [start, end)
			aggLists = append(aggLists, &aggUtil{start: start, end: end, cols: countList.Cols})
			break
		}

		if end > countList.Start {
			//[countList.Start, end)
			aggLists = append(aggLists, &aggUtil{start: countList.Start, end: end, cols: countList.Cols})
			end = countList.Start
			continue
		} else {
			continue
		}
	}

	var (
		ewg    multierror.Group
		res    = make([][]bson.M, len(aggLists))
		result = make([]bson.M, 0)
	)

	for i := range aggLists {
		i := i
		aggList := aggLists[i]
		ewg.Go(func() error {
			var aggRes []bson.M

			// 防止有分页需求的脚本
			if req.Index == 0 && req.Limit == 0 {
				req.Limit = math.MaxInt64
			}

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.start, EndEpoch: aggList.end, Addr: req.Addr, Addrs: req.Addrs, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, Cids: req.Cids, ID: req.ID, Sort: req.Sort, To: req.To, Skip: req.Index * req.Limit, Limit: req.Limit, ExpirationStartEpoch: req.ExpirationStartEpoch, ExpirationEndEpoch: req.ExpirationEndEpoch, SectorSize: req.SectorSize, TransferType: req.TransferType}, string(aggregator))
			if err != nil {
				return err
			}

			for _, col := range aggList.cols.Cols {
				if col != nil && col.Name() == tableName {
					cur, err := col.Aggregate(ctx, pipe)
					if err != nil {
						return err
					}

					err = cur.All(ctx, &aggRes)
					if err != nil {
						return err
					}

					res[i] = aggRes
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		result = append(result, res[i]...)
	}

	return result, nil
}

// MultiTraversalQuery 并发遍历所有库查询, 取一个结果
func MultiTraversalQuery(ctx context.Context, pipe interface{}, countLists []CountUtil, tableName string) ([]bson.M, error) {
	var (
		ewg    multierror.Group
		result = make([]bson.M, 0)
		lock   sync.RWMutex
	)

	// 优先查询tmp和formal，未查到再并发查询冷库
	priorityLists := make([]CountUtil, 0)
	delayedLists := make([]CountUtil, 0)
	for _, countList := range countLists {
		if countList.DType == smodel.Formal || countList.DType == smodel.Tmp {
			priorityLists = append(priorityLists, countList)
		} else {
			delayedLists = append(delayedLists, countList)
		}
	}

	for _, countList := range priorityLists {
		for _, col := range countList.Cols.Cols {
			if col != nil && col.Name() == tableName {
				cur, err := col.Aggregate(ctx, pipe)
				if err != nil {
					return nil, err
				}

				err = cur.All(ctx, &result)
				if err != nil {
					return nil, err
				}

				if len(result) > 0 {
					return result, nil
				}
			}
		}
	}

	for i := range delayedLists {
		i := i
		countList := delayedLists[i]

		ewg.Go(func() error {
			var res []bson.M

			for _, col := range countList.Cols.Cols {
				if col != nil && col.Name() == tableName {
					lock.RLock()
					if len(result) > 0 {
						// 有库已经查出来了
						lock.RUnlock()
						return nil
					}
					lock.RUnlock()

					cur, err := col.Aggregate(ctx, pipe)
					if err != nil {
						return err
					}

					err = cur.All(ctx, &res)
					if err != nil {
						return err
					}

					if len(res) > 0 {
						lock.Lock()
						result = res
						lock.Unlock()
						return nil
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

// MultiUnionQuery 并发遍历所有库查询, 所有结果都要
func MultiUnionQuery(ctx context.Context, pipe interface{}, countLists []CountUtil, tableName string) ([]bson.M, error) {
	var (
		ewg    multierror.Group
		res    = make([][]bson.M, len(countLists))
		result = make([]bson.M, 0)
	)

	// todo: 所有库都查询一次，最终查询时间为最慢库的查询时间，费时
	// 先查询formal，再并发查询冷库？
	for i := range countLists {
		i := i
		countList := countLists[i]
		ewg.Go(func() error {
			var aggRees []bson.M

			for _, col := range countList.Cols.Cols {
				if col != nil && col.Name() == tableName {
					cur, err := col.Aggregate(ctx, pipe)
					if err != nil {
						return err
					}

					err = cur.All(ctx, &aggRees)
					if err != nil {
						return err
					}
					res[i] = aggRees
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		result = append(result, res[i]...)
	}

	return result, nil
}

// actor message
func MultiBiSearch(ctx context.Context, indexReq, limitReq int64, countUtils []CountUtil, aggregator, countAgg []byte,
	req model.CommonReq, tableName string, ptype Ptype) ([]bson.M, error) {
	sum := int64(0)
	result := []bson.M{}
	for i := range countUtils {
		tmp := int64(0)
		switch ptype {
		case ActorStates:
			tmp = countUtils[i].ActorStates
		case ActorMethodStates:
			tmp = countUtils[i].ActorMethodStates
		case MinedStates:
			tmp = countUtils[i].MinedStates
		case ActorTransferStates:
			tmp = countUtils[i].ActorTransferStates
		default:
			return nil, fmt.Errorf("not support bi search")
		}
		if sum+tmp <= indexReq {
			sum += tmp
			continue
		}

		remainSum := indexReq - sum
		// 找到最大的一个start使得 start->countUtil.end的个数和>剩下的sum个数
		start, rangeSum, startSum, err := BiSearch(ctx, remainSum, countUtils[i], countAgg, tableName, req)
		if err != nil {
			log.Errorf("bi search failed : %w", err)
			return nil, err
		}
		// 从start+1开始，倒序统计从start+1->countUtils.start的 limit+剩下的sum个数的
		res, err := GetDetail(ctx, start+1, limitReq+(startSum-(rangeSum-remainSum)), countUtils[i], aggregator, tableName, req)
		if err != nil {
			return nil, err
		}
		result = append(result, res[int(startSum-(rangeSum-remainSum)):]...)
		// 从下一个countUtil开始
		countUtilsIdx := i + 1
		for len(result) < int(limitReq) && countUtilsIdx < len(countUtils) {
			tmp := int64(0)
			switch ptype {
			case ActorStates:
				tmp = countUtils[i].ActorStates
			case ActorMethodStates:
				tmp = countUtils[i].ActorMethodStates
			case MinedStates:
				tmp = countUtils[i].MinedStates
			case ActorTransferStates:
				tmp = countUtils[i].ActorTransferStates
			default:
				return nil, fmt.Errorf("not support bi search")
			}

			if tmp != 0 {
				res, err := GetDetail(ctx, countUtils[countUtilsIdx].End, limitReq-int64(len(result)), countUtils[countUtilsIdx], aggregator, tableName, req)
				if err != nil {
					return nil, err
				}
				result = append(result, res...)
			}
			countUtilsIdx++
		}
		break
	}

	return result, nil
}

func BiSearch(ctx context.Context, targetSum int64, countUtil CountUtil, countAgg []byte, tableName string, req model.CommonReq) (int64, int64, int64, error) {
	l, r := countUtil.Start, countUtil.End
	res := countUtil.Start
	col1 := &mongo.Collection{}
	for i := range countUtil.Cols.Cols {
		if countUtil.Cols.Cols[i] != nil && countUtil.Cols.Cols[i].Name() == tableName {
			col1 = countUtil.Cols.Cols[i]
		}
	}
	resCount := int64(0)
	startSum := int64(0)
	for l < r {
		mid := (l + r) / 2
		rangeSum, err := CommonCount(ctx, col1, req, countAgg, mid, countUtil.End)
		if err != nil {
			log.Errorf("count address messages failed: %w", err)
			return 0, 0, 0, err
		}

		if rangeSum > targetSum {
			res = mid
			resCount = rangeSum

			l = mid + 1
		} else {
			r = mid
		}
	}
	startSum, err := CommonCount(ctx, col1, req, countAgg, res, res+1)
	return res, resCount, startSum, err
}

func CommonCount(ctx context.Context, col *mongo.Collection, req model.CommonReq, countAgg []byte, st, ed int64) (int64, error) {
	pipe, err := util.Parse(model.Ctx{StartEpoch: st, EndEpoch: ed, Addr: req.Addr, Sort: -1,
		Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, ID: req.ID, To: req.To, Addrs: req.Addrs, TransferType: req.TransferType}, string(countAgg))
	if err != nil {
		return 0, err
	}
	var countRes []model.CountRes
	cur, err := col.Aggregate(ctx, pipe)
	if err != nil {
		return 0, err
	}
	err = cur.All(ctx, &countRes)
	if err != nil {
		return 0, err
	}
	res := int64(0)
	if len(countRes) != 0 {
		res = countRes[0].Count
	}
	return res, nil
}

func GetDetail(ctx context.Context, end, limit int64, countUtil CountUtil, aggregator []byte, tableName string, req model.CommonReq) ([]bson.M, error) {
	col1 := &mongo.Collection{}
	for i := range countUtil.Cols.Cols {
		if countUtil.Cols.Cols[i] != nil && countUtil.Cols.Cols[i].Name() == tableName {
			col1 = countUtil.Cols.Cols[i]
		}
	}
	pipe, err := util.Parse(model.Ctx{StartEpoch: countUtil.Start, EndEpoch: end, Addr: req.Addr, Sort: -1, Limit: limit,
		Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, ID: req.ID, To: req.To, Addrs: req.Addrs, TransferType: req.TransferType}, string(aggregator))
	if err != nil {
		log.Errorf("get detail parse pipe failed: %w", err)
		return nil, err
	}

	cur, err := col1.Aggregate(ctx, pipe)
	if err != nil {
		log.Errorf("get detail agg failed: %w", err)
		return nil, err
	}
	var aggRes []bson.M
	err = cur.All(ctx, &aggRes)
	if err != nil {
		log.Errorf("get detail all failed: %w", err)
		return nil, err
	}
	return aggRes, nil
}

// MultiPagingQueryForDeal 对deal分页查询，
func MultiPagingQueryForDeal(ctx context.Context, indexReq, limitReq int64, ptype Ptype, countUtils []CountUtil, aggregator []byte, req model.CommonReq, tableName string) ([]bson.M, error) {
	if ptype != DealState {
		return nil, fmt.Errorf("use MultiPagingQueryForDeal for non deal, ptype: %v", ptype)
	}

	// skip、limit
	skipTag := func(skip, count int64) bool {
		return skip < count
	}
	limitTag := func(requestTotalCount, count int64) bool {
		return requestTotalCount <= count
	}

	// todo: skip=0 & limit=0 获取全量？
	skip := int64(0)
	limit := int64(math.MaxInt64)
	requestTotalCount := int64(math.MaxInt64)

	if indexReq > 0 || limitReq > 0 {
		skip = indexReq * limitReq
		limit = limitReq
		requestTotalCount = skip + limit
	}

	segmentLists := make([]*segmentUtil, 0)

	// 锁定库
	for _, countlist := range countUtils {
		totalCount := int64(0)
		totalCount = countlist.DealState

		segment := &segmentUtil{start: countlist.Start, end: countlist.End, Cols: countlist.Cols,
			BlockStates:               countlist.BlockStates,
			BlockMethodStates:         countlist.BlockMethodStates,
			ActorStates:               countlist.ActorStates,
			ActorMethodStates:         countlist.ActorMethodStates,
			ActorTransferStates:       countlist.ActorTransferStates,
			MinedStates:               countlist.MinedStates,
			LargeAmountTransferStates: countlist.LargeAmountTransferStates,
			DealState:                 countlist.DealState,
			DealActorStates:           countlist.DealActorStates,
			TipSetStates:              countlist.TipSetStates,
		}

		skipflag := skipTag(skip, totalCount)
		limitflag := limitTag(requestTotalCount, totalCount)
		if skipflag && limitflag {
			segment.skip = skip
			segment.limit = limit
			segmentLists = append(segmentLists, segment)
			break
		}
		if skipflag && !limitflag {
			segment.skip = skip
			segment.limit = limit
			segmentLists = append(segmentLists, segment)
			skip = 0
			limit = requestTotalCount - totalCount
			requestTotalCount = requestTotalCount - totalCount // skip 1, limit 5  tmp 8
			continue
		}
		if !skipflag {
			skip = skip - totalCount
			requestTotalCount = skip + limit
			continue
		}
	}

	// 直接根据范围拿分页内容，不用skip
	aggLists := make([]*aggUtil, 0)
	for _, segmentList := range segmentLists {
		start := segmentList.start
		end := segmentList.end
		skip := segmentList.skip
		limit := segmentList.limit

		if segmentList.skip > 0 {
			end = segmentList.end - segmentList.skip
			start = end - (segmentList.limit)
			skip = 0
			limit = segmentList.limit
		}

		aggLists = append(aggLists, &aggUtil{
			start: start,
			end:   end,
			skip:  skip,
			limit: limit,
			cols:  segmentList.Cols,
			count: segmentList.DealState,
		})
	}

	// concurrent agg
	var (
		res    = make([][]bson.M, len(aggLists))
		result = make([]bson.M, 0)
		ewg    multierror.Group
	)

	lim := limiter.New(16)

	for i := range aggLists {
		i := i
		aggList := aggLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				return nil
			}

			defer func() {
				lim.Release(ctx)
			}()

			var aggRes []bson.M

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.start, EndEpoch: aggList.end, Start: aggList.start, End: aggList.end, Skip: aggList.skip, Limit: aggList.limit, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, ID: req.ID, Sort: req.Sort, To: req.To, Addrs: req.Addrs, Addr: req.Addr}, string(aggregator)) // todo: methodName
			if err != nil {
				return err
			}

			for _, col := range aggList.cols.Cols {
				if col != nil && col.Name() == tableName {
					cur, err := col.Aggregate(ctx, pipe)
					if err != nil {
						return err
					}

					err = cur.All(ctx, &aggRes)
					if err != nil {
						return err
					}

					res[i] = aggRes

					return nil
				}
			}

			return fmt.Errorf("no collection: %v", tableName)
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	for i := 0; i < len(res); i++ {
		result = append(result, res[i]...)
	}

	return result, nil
}
