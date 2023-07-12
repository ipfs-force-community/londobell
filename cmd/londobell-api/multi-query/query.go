package multiquery

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

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

func GetStartEpochForDeal(ctx context.Context, cols common.Collections) (int64, error) {
	var res []struct {
		Epoch int64
	}

	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetStartEpochForDealAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "DealProposal" {
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

	return 0, fmt.Errorf("no DealProposal collection")
}

func GetStartDealID(ctx context.Context, cols common.Collections, startEpoch int64) (uint64, error) {
	var res []struct {
		StartDealID uint64
	}

	pipe, err := util.Parse(model.Ctx{StartEpoch: startEpoch}, string(monitor.GetStartDealIDAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "DealProposal" {
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

			return res[0].StartDealID, nil
		}
	}

	return 0, fmt.Errorf("no DealProposal collection")
}

// max dealID+1
func GetEndDealID(ctx context.Context, cols common.Collections, startEpoch int64) (uint64, error) {
	var res []struct {
		EndDealID uint64
	}

	pipe, err := util.Parse(model.Ctx{StartEpoch: startEpoch}, string(monitor.GetEndDealIDAggregator()))
	if err != nil {
		return 0, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "DealProposal" {
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

			return res[0].EndDealID + 1, nil
		}
	}

	return 0, fmt.Errorf("no DealProposal collection")
}

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
			totalCount = countlist.BlockMethodStates
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
			skip, limit := segmentList.skip, segmentList.limit
			requestTotalCount := skip + limit

			sort.Slice(segmentList.BlockStates, func(i, j int) bool {
				return segmentList.BlockStates[i].StartEpoch > segmentList.BlockStates[j].StartEpoch
			})

			for _, bs := range segmentList.BlockStates {
				count := bs.Count

				agg := &aggUtil{start: int64(bs.StartEpoch), end: int64(bs.EndEpoch), cols: segmentList.Cols}
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
					requestTotalCount = requestTotalCount - count // skip 1, limit 5  tmp 8
					continue
				}
				if !skipflag {
					skip = skip - count
					requestTotalCount = skip + limit
					continue
				}
			}
		}
	case BlockMethodStates:
		for _, segmentList := range segmentLists {
			aggLists = append(aggLists, &aggUtil{
				start: segmentList.start,
				end:   segmentList.end,
				skip:  segmentList.skip,
				limit: segmentList.limit,
				cols:  segmentList.Cols,
				count: segmentList.BlockMethodStates,
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

		//for _, segmentList := range segmentLists {
		//	skip, limit := segmentList.skip, segmentList.limit
		//	requestTotalCount := skip + limit
		//
		//	sort.Slice(segmentList.DealActorStates, func(i, j int) bool {
		//		return segmentList.DealActorStates[i].StartDealID > segmentList.DealActorStates[j].StartDealID
		//	})
		//
		//	for _, das := range segmentList.DealActorStates {
		//		count := das.Count
		//
		//		agg := &aggUtil{start: int64(das.StartDealID), end: int64(das.EndDealID), cols: segmentList.Cols}
		//		skipflag := skipTag(skip, count)
		//		limitflag := limitTag(requestTotalCount, count)
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
		//			limit = requestTotalCount - count
		//			requestTotalCount = requestTotalCount - count // skip 1, limit 5  tmp 8
		//			continue
		//		}
		//		if !skipflag {
		//			skip = skip - count
		//			requestTotalCount = skip + limit
		//			continue
		//		}
		//	}
		//}
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

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.start, EndEpoch: aggList.end, Addr: req.Addr, Addrs: req.Addrs, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, Cids: req.Cids, ID: req.ID, Sort: req.Sort, To: req.To, Skip: req.Index * req.Limit, Limit: req.Limit}, string(aggregator))
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
