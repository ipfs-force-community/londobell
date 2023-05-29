package multiquery

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

func GetFinalHeight(ctx context.Context, cols Collections) (abi.ChainEpoch, error) {
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

func GetStateFinalHeight(ctx context.Context, cols Collections) (abi.ChainEpoch, error) {
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

// MultiPagingQuery 对多库进行分页查询  todo: 磁盘state更改会影响dbStates（map）吗？
func MultiPagingQuery(ctx context.Context, indexReq, limitReq int64, countUtils []CountUtil, aggregator []byte, req model.CommonReq, tableName string) ([]bson.M, error) {
	qlog := log.With("multi-query", "MultiPagingQuery")

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

	aggLists := make([]*aggUtil, 0)
	for i, countlist := range countUtils {
		skipflag := skipTag(skip, countlist.Count)
		limitflag := limitTag(requestTotalCount, countlist.Count)
		if skipflag && limitflag {
			// 最后一次
			qlog.Infof("skipflag && limitflag, index: %v, skip: %v, limit: %v, requestTotalCount: %v, start: %v, end: %v", i, skip, limit, requestTotalCount, countlist.Start, countlist.End)

			aggLists = append(aggLists, &aggUtil{startEpoch: countlist.Start, endEpoch: countlist.End, skip: skip, limit: limit, cols: countlist.Cols})
			break
		}
		if skipflag && !limitflag {
			qlog.Infof("skipflag && !limitflag, index: %v, skip: %v, limit: %v, requestTotalCount: %v", i, skip, limit, requestTotalCount)

			aggLists = append(aggLists, &aggUtil{startEpoch: countlist.Start, endEpoch: countlist.End, skip: skip, limit: limit, cols: countlist.Cols})
			skip = 0
			limit = requestTotalCount - countlist.Count
			requestTotalCount = requestTotalCount - countlist.Count // skip 1, limit 5  tmp 8
			continue
		}
		if !skipflag {
			qlog.Infof("!skipflag, index: %v, skip: %v, limit: %v, requestTotalCount: %v", i, skip, limit, requestTotalCount)

			skip = skip - countlist.Count
			requestTotalCount = skip + limit
			continue
		}
	}

	qlog.Infof("get aggLists done!!")

	// concurrent agg
	var (
		res    = make([][]bson.M, len(aggLists))
		result = make([]bson.M, 0)
		ewg    multierror.Group
	)

	for i := range aggLists {
		i := i
		aggList := aggLists[i]
		ewg.Go(func() error {
			start := time.Now()
			var aggRes []bson.M

			//// todo: req.Addr 和 /*req.Cid*/ 要请求多个等价
			//if req.Addr != "" && aggregator ==  {
			//	//pipe1...
			//}

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.startEpoch, EndEpoch: aggList.endEpoch, Skip: aggList.skip, Limit: aggList.limit, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, ID: req.ID, Sort: req.Sort, To: req.To, Addrs: req.Addrs, Addr: req.Addr}, string(aggregator)) // todo: methodName
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

			qlog.Infof("agg successfully, agglist: %+v spent: %v", aggList, time.Now().Sub(start))
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

// MultiRangeQuery 根据epoch范围定位到某些库查询
func MultiRangeQuery(ctx context.Context, startEpoch, endEpoch int64, countUtils []CountUtil, aggregator []byte, req model.CommonReq, tableName string) ([]bson.M, error) {
	qlog := log.With("multi-query", "MultiRangeQuery")

	start := startEpoch
	end := endEpoch
	aggLists := make([]*aggUtil, 0)
	for i := range countUtils {
		i := i
		countList := countUtils[i]
		if start >= countList.Start {
			// [start, end)
			aggLists = append(aggLists, &aggUtil{startEpoch: start, endEpoch: end, cols: countList.Cols})
			break
		}

		if end > countList.Start {
			//[countList.Start, end)
			aggLists = append(aggLists, &aggUtil{startEpoch: countList.Start, endEpoch: end, cols: countList.Cols})
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
			start := time.Now()
			var aggRes []bson.M

			// 防止有分页需求的脚本
			if req.Index == 0 && req.Limit == 0 {
				req.Limit = math.MaxInt64
			}

			pipe, err := util.Parse(model.Ctx{StartEpoch: aggList.startEpoch, EndEpoch: aggList.endEpoch, Addr: req.Addr, Addrs: req.Addrs, Method: req.Method, MethodName: req.MethodName, Cid: req.Cid, Cids: req.Cids, ID: req.ID, Sort: req.Sort, To: req.To, Skip: req.Index * req.Limit, Limit: req.Limit}, string(aggregator))
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

			qlog.Infof("agg successfully for block, aggulist %+v spent %v", aggList, time.Now().Sub(start))
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
		if countList.Tmp || countList.Formal {
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
