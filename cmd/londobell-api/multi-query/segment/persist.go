package segment

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
)

type AddUpState struct {
	State State
	lk    sync.RWMutex
}

func NewAddUpState(state State) *AddUpState {
	return &AddUpState{
		State: state,
	}
}

func (a *AddUpState) GetState() State {
	a.lk.RLock()
	defer a.lk.RUnlock()

	return a.State
}

func (a *AddUpState) UpdateState(state State) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State = state
}

type PersistState struct {
	Dsn       string
	Start     int64
	End       int64
	Count     int64
	NextStart int64
}

// 累加
func (s *Segment) AddUpBlockState(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "BlockState")

	state := addUpState.GetState()

	dbState := state.GetDBState()
	blockStates := state.GetBlockStates()

	newBlockStates := make([]model.SegmentState, 0)

	startEpoch, dsn, interval, dType := int64(dbState.StartEpoch), dbState.Dsn, dbState.Interval, dbState.DType

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup BlockState successfully between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
	}()

	sort.Slice(blockStates, func(i, j int) bool {
		return blockStates[i].StartEpoch > blockStates[j].StartEpoch
	})

	var endEpoch int64
	if len(blockStates) == 0 {
		endEpoch = startEpoch
	} else {
		endEpoch = int64(blockStates[0].EndEpoch)
	}

	if nextEndEpoch <= endEpoch {
		rlog.Infof("skip for nextEndEpoch %v <= endEpoch %v", nextEndEpoch, endEpoch)
		return nil
	}

	var (
		todoBlockStates   = make([]PersistState, 0)
		initialBlockState PersistState
		initialEpoch      int64
	)

	length := len(blockStates)
	if length == 0 {
		initialEpoch = startEpoch
		initialBlockState = PersistState{Dsn: dsn, Start: startEpoch, NextStart: startEpoch}
	} else {
		start, end, count := int64(blockStates[0].StartEpoch), int64(blockStates[0].EndEpoch), blockStates[0].Count
		if end-start == interval {
			// new next blockState
			initialEpoch = end
			initialBlockState = PersistState{Dsn: dsn, Start: end, NextStart: end}
			newBlockStates = blockStates[:]
		} else {
			initialEpoch = start
			initialBlockState = PersistState{Dsn: dsn, Start: start, Count: count, NextStart: end}
			newBlockStates = blockStates[1:]
		}
	}

	first := true
	for initialEpoch < nextEndEpoch {
		end := initialEpoch + interval
		if end > nextEndEpoch {
			end = nextEndEpoch
		}

		if first {
			initialBlockState.End = end
			todoBlockStates = append(todoBlockStates, initialBlockState)
			first = false
		} else {
			todoBlockStates = append(todoBlockStates, PersistState{Dsn: dsn, Start: initialEpoch, End: end, NextStart: initialEpoch})
		}

		initialEpoch = end
	}

	lim := limiter.New(s.opts.BatchInsertLimit)
	var lk sync.Mutex

	var ewg multierror.Group
	for _, todoBlockState := range todoBlockStates {
		todoBlockState := todoBlockState
		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				return nil
			}

			defer func() {
				lim.Release(ctx)
			}()

			starttime := time.Now()
			start, end, dsn, nextStart := todoBlockState.Start, todoBlockState.End, todoBlockState.Dsn, todoBlockState.NextStart
			defer func() {
				rlog.Infof("addup BlockMsgsCount successfully between %v and %v, count: %v, elapsed: %v", start, end, todoBlockState.Count, time.Now().Sub(starttime).String())
			}()

			blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: nextStart}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}}, {Key: "IsBlock", Value: true}}

			for _, col := range cols.Cols {
				if col != nil && col.Name() == "ExecTrace" {
					count, err := col.CountDocuments(ctx, blockFilter)
					if err != nil {
						return fmt.Errorf("count for block messages failed: %w", err)
					}

					todoBlockState.Count += count

					// todo: 测试插入成功
					err = s.db.FindOneAndUpdate(ctx, "BlockState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, start)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: start}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: end}, {Key: "Count", Value: todoBlockState.Count}}}})
					if err != nil {
						return fmt.Errorf("update blockstate failed: %w", err)
					}

					lk.Lock()
					newBlockStates = append(newBlockStates, model.SegmentState{ID: fmt.Sprintf("%v-%v", dsn, start), Dsn: dsn, StartEpoch: abi.ChainEpoch(start), EndEpoch: abi.ChainEpoch(end), Count: todoBlockState.Count})
					lk.Unlock()

					return nil
				}
			}

			return fmt.Errorf("no ExecTrace collections")
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	dbState.EndEpoch = abi.ChainEpoch(nextEndEpoch)
	err := s.db.FindOneAndUpdate(ctx, "DBState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dbState.StartEpoch)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: dbState.StartEpoch}, {Key: "DType", Value: dType}, {Key: "Interval", Value: interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: dbState.EndEpoch}}}})
	if err != nil {
		return fmt.Errorf("update dbstate failed: %w", err)
	}

	state.SetDBState(dbState)

	if err := state.SetBlockStates(newBlockStates); err != nil {
		return err
	}

	addUpState.UpdateState(state)

	return nil
}

//func (s *Segment) AddUpDealState(ctx context.Context, log *zap.SugaredLogger, nextDealID int64, addUpState *AddUpState, cols common.Collections) error {
//	rlog := log.With("AddUp", "DealState")
//
//	state := addUpState.GetState()
//
//	dealState := state.GetDealState()
//
//	newDealStates := make([]model.DealState, 0)
//
//	startDealID, dsn, interval, endDealID, count, dType := dealState.StartDealID, dealState.Dsn, dealState.Interval, dealState.EndDealID, dealState.Count, dealState.DType
//
//	starttime := time.Now()
//	defer func() {
//		rlog.Infof("addup DealActorState successfully between %v and %v, elapsed: %v", startDealID, nextDealID, time.Now().Sub(starttime).String())
//	}()
//
//	if uint64(nextDealID) <= endDealID {
//		rlog.Infof("skip for nextDealID %v <= endDealID %v", nextDealID, endDealID)
//		return nil
//	}
//
//	var (
//		todoDealStates   = make([]PersistState, 0)
//		initialDealState PersistState
//		initialDealID    int64
//	)
//
//	if interval == model.NoneInterval {
//		todoDealStates = append(todoDealStates, PersistState{Dsn: dsn, Start: int64(startDealID), End: nextDealID, NextStart: int64(endDealID), Count: count})
//	} else {
//		if int64(endDealID-startDealID) == interval {
//			initialDealID = int64(endDealID)
//			initialDealState = PersistState{Dsn: dsn, Start: int64(endDealID), NextStart: int64(endDealID)}
//		} else {
//			initialDealID = int64(startDealID)
//			initialDealState = PersistState{Dsn: dsn, Start: int64(startDealID), Count: dealState.Count, NextStart: int64(endDealID)}
//		}
//
//		first := true
//		for initialDealID < nextDealID {
//			end := initialDealID + interval
//			if end > nextDealID {
//				end = nextDealID
//			}
//
//			if first {
//				initialDealState.End = end
//				todoDealStates = append(todoDealStates, initialDealState)
//				first = false
//			} else {
//				todoDealStates = append(todoDealStates, PersistState{Dsn: dsn, Start: initialDealID, End: end, NextStart: initialDealID})
//			}
//
//			initialDealID = end
//		}
//	}
//
//	lim := limiter.New(s.opts.BatchInsertLimit)
//	var lk sync.Mutex
//
//	var ewg multierror.Group
//	for _, todoDealState := range todoDealStates {
//		todoDealState := todoDealState
//		ewg.Go(func() error {
//			if !lim.Acquire(ctx) {
//				return nil
//			}
//
//			defer func() {
//				lim.Release(ctx)
//			}()
//
//			starttime := time.Now()
//			start, end, dsn, nextStart := todoDealState.Start, todoDealState.End, todoDealState.Dsn, todoDealState.NextStart
//			defer func() {
//				rlog.Infof("addup DealActorState successfully between %v and %v, count: %v, elapsed: %v", start, end, todoDealState.Count, time.Now().Sub(starttime).String())
//			}()
//
//			var res []struct {
//				_id struct {
//					Provider string
//					Client   string
//				}
//				Count int64
//			}
//
//			pipe, err := util.Parse(model.Ctx{Start: nextStart, End: end}, dealActorJS)
//			if err != nil {
//				return err
//			}
//
//			for _, col := range cols.Cols {
//				if col != nil && col.Name() == "DealProposal" {
//					count := int64(0)
//					cur, err := col.Aggregate(ctx, pipe)
//					if err != nil {
//						return err
//					}
//
//					if err := cur.All(ctx, res); err != nil {
//						return err
//					}
//
//					todoDealState.Count += count
//
//					err = s.db.FindOneAndUpdate(ctx, "DealState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, start)}, {Key: "Dsn", Value: dsn}, {Key: "StartDealID", Value: start}, {Key: "DType", Value: dType}, {Key: "Interval", Value: interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndDealID", Value: end}, {Key: "Count", Value: todoDealState.Count}}}})
//					if err != nil {
//						return fmt.Errorf("update dealActorState failed: %w", err)
//					}
//
//					lk.Lock()
//					newDealStates = append(newDealStates, model.DealState{ID: fmt.Sprintf("%v-%v", dsn, start), DType: dealState.DType, Interval: interval, StartDealID: uint64(start), EndDealID: uint64(end), Count: todoDealState.Count})
//					lk.Unlock()
//
//					return nil
//				}
//			}
//
//			return fmt.Errorf("no DealProposal collections")
//		})
//	}
//
//	if err := ewg.Wait(); err != nil {
//		return err
//	}
//
//	dealState.EndDealID = uint64(nextDealID)
//	err := s.db.FindOneAndUpdate(ctx, "DealState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dealState.StartDealID)}, {Key: "Dsn", Value: dsn}, {Key: "StartDealID", Value: dealState.StartDealID}, {Key: "DType", Value: dealState.DType}, {Key: "Interval", Value: dealState.Interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndDealID", Value: dealState.EndDealID}}}})
//	if err != nil {
//		return fmt.Errorf("update dbstate failed: %w", err)
//	}
//
//	state.SetDealState(dealState)
//
//	if err := state.SetDealState(newDealStates); err != nil {
//		return err
//	}
//
//	addUpState.UpdateState(state)
//
//	return nil
//}

// DealActorState 需读取所有actor的dealcount，消耗内存较大，且目前看来无速度优化续期
//func (s *Segment) AddUpDealActorState(ctx context.Context, log *zap.SugaredLogger, nextDealID int64, addUpState *AddUpState, cols common.Collections) error {
//	rlog := log.With("AddUp", "DealActorState")
//
//	state := addUpState.GetState()
//
//	dealState := state.GetDealState()
//	allDealActorStates := state.GetAllDealActorStates()
//
//	newDealActorStates := make([]model.SegmentDealState, 0)
//
//	startDealID, dsn, interval := dealState.StartDealID, dealState.Dsn, dealState.Interval
//
//	starttime := time.Now()
//	defer func() {
//		rlog.Infof("addup DealActorState successfully between %v and %v, elapsed: %v", startDealID, nextDealID, time.Now().Sub(starttime).String())
//	}()
//
//	sort.Slice(allDealActorStates, func(i, j int) bool {
//		return allDealActorStates[i].StartDealID > allDealActorStates[j].StartDealID
//	})
//
//	var endDealID uint64
//	if len(allDealActorStates) == 0 {
//		endDealID = startDealID
//	} else {
//		endDealID = allDealActorStates[0].EndDealID
//	}
//
//	if uint64(nextDealID) <= endDealID {
//		rlog.Infof("skip for nextDealID %v <= endDealID %v", nextDealID, endDealID)
//		return nil
//	}
//
//	var (
//		todoDealActorStates   = make([]model.SegmentDealState, 0)
//		initialDealActorState model.SegmentDealState
//		initialDealID         uint64
//	)
//
//	length := len(allDealActorStates)
//	if length == 0 {
//		initialDealID = startDealID
//		initialDealActorState = model.SegmentDealState{Dsn: dsn, StartDealID: startDealID}
//	} else {
//		start, end, count := allDealActorStates[0].StartDealID, allDealActorStates[0].EndDealID, allDealActorStates[0].Count
//		if int64(end-start) == interval {
//			// new next dealActorState
//			initialDealID = end
//			initialDealActorState = model.SegmentDealState{Dsn: dsn, StartDealID: end}
//			newDealActorStates = allDealActorStates[:]
//		} else if int64(end-start) < interval || interval == model.NoneInterval {
//			initialDealID = start
//			initialDealActorState = model.SegmentDealState{Dsn: dsn, StartDealID: start, Count: count}
//			newDealActorStates = allDealActorStates[1:]
//		}
//	}
//
//	if interval == model.NoneInterval {
//		initialDealActorState.EndDealID = uint64(nextDealID)
//		todoDealActorStates = append(todoDealActorStates, initialDealActorState)
//	} else {
//		first := true
//		for initialDealID < uint64(nextDealID) {
//			end := initialDealID + uint64(interval)
//			if end > uint64(nextDealID) {
//				end = uint64(nextDealID)
//			}
//
//			if first {
//				initialDealActorState.EndDealID = end
//				todoDealActorStates = append(todoDealActorStates, initialDealActorState)
//				first = false
//			} else {
//				todoDealActorStates = append(todoDealActorStates, model.SegmentDealState{Dsn: dsn, StartDealID: initialDealID, EndDealID: end})
//			}
//
//			initialDealID = end
//		}
//	}
//
//	newDealActorStates = append(newDealActorStates, todoDealActorStates...)
//	lim := limiter.New(s.opts.BatchInsertLimit)
//
//	var ewg multierror.Group
//	for _, todoDealActorState := range todoDealActorStates {
//		todoDealActorState := todoDealActorState
//		ewg.Go(func() error {
//			if !lim.Acquire(ctx) {
//				return nil
//			}
//
//			defer func() {
//				lim.Release(ctx)
//			}()
//
//			starttime := time.Now()
//			start, end, dsn := todoDealActorState.StartDealID, todoDealActorState.EndDealID, todoDealActorState.Dsn
//			defer func() {
//				rlog.Infof("addup DealActorState successfully between %v and %v, count: %v, elapsed: %v", start, end, todoDealActorState.Count, time.Now().Sub(starttime).String())
//			}()
//
//			var res []struct {
//				_id struct {
//					Provider string
//					Client   string
//				}
//				Count int64
//			}
//
//			pipe, err := util.Parse(model.Ctx{Start: int64(start), End: int64(end)}, monitor.GetCountForDealsByProviderAndClientAggregator())
//			if err != nil {
//				return err
//			}
//
//			api := fullnode.API.GetAppropriateAPI()
//			for _, col := range cols.Cols {
//				if col != nil && col.Name() == "DealProposal" {
//					count := int64(0)
//					cur, err := col.Aggregate(ctx, pipe)
//					if err != nil {
//						return err
//					}
//
//					if err := cur.All(ctx, res); err != nil {
//						return err
//					}
//
//					dealCountMap := make(map[string]int64)
//					for _, r := range res {
//						provider, err := address.NewFromString(buildnet.NetPrefix + r._id.Provider)
//						if err != nil {
//							return err
//						}
//
//						providerID, err := api.StateLookupID(ctx, provider, types.EmptyTSK)
//						if err != nil {
//							return err
//						}
//
//						client, err := address.NewFromString(buildnet.NetPrefix + r._id.Client)
//						if err != nil {
//							return err
//						}
//
//						clientID, err := api.StateLookupID(ctx, client, types.EmptyTSK)
//						if err != nil {
//							return err
//						}
//
//						dealCountMap[providerID.String()[1:]] += r.Count
//						dealCountMap[clientID.String()[1:]] += r.Count
//					}
//
//					for actorID, count := range dealCountMap {
//						todoDealActorState.ActorID
//					}
//
//					todoDealActorState.Count += count
//
//					err = s.db.FindOneAndUpdate(ctx, "DealActorState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, start)}, {Key: "Dsn", Value: dsn}, {Key: "StartDealID", Value: start}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndDealID", Value: todoDealActorState.EndDealID}, {Key: "Count", Value: todoDealActorState.Count}}}})
//					if err != nil {
//						return fmt.Errorf("update dealActorState failed: %w", err)
//					}
//
//					return nil
//				}
//			}
//
//			return fmt.Errorf("no DealProposal collections")
//		})
//	}
//
//	if err := ewg.Wait(); err != nil {
//		return err
//	}
//
//	dealState.EndDealID = uint64(nextDealID)
//	err := s.db.FindOneAndUpdate(ctx, "DealState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dealState.StartDealID)}, {Key: "Dsn", Value: dsn}, {Key: "StartDealID", Value: dealState.StartDealID}, {Key: "DType", Value: dealState.DType}, {Key: "Interval", Value: dealState.Interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndDealID", Value: dealState.EndDealID}}}})
//	if err != nil {
//		return fmt.Errorf("update dbstate failed: %w", err)
//	}
//
//	state.SetDealState(dealState)
//
//	if err := state.SetDealActorStates(newDealActorStates); err != nil {
//		return err
//	}
//
//	addUpState.UpdateState(state)
//
//	return nil
//}

func (s *Segment) GetState(ctx context.Context, dsn string) (*State, bool, error) {
	dbState, found, err := s.GetDBState(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	if !found {
		return nil, false, nil
	}

	blockStates, err := s.GetBlockStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	blockMethodStates, err := s.GetAllBlockMethodStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	//if !found {
	//	// todo: 返回0值
	//	blockMethodStates = []model.SegmentState{{Dsn: dbState.Dsn, StartEpoch: dbState.StartEpoch, EndEpoch: dbState.EndEpoch}}
	//}

	actorStates, err := s.GetAllActorStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	actorMethodStates, err := s.GetAllActorMethodStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	actorTransferStates, err := s.GetAllActorTransferStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	minedStates, err := s.GetAllMinedStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	largeAmountTransferStates, err := s.GetLargeAmountTransferStates(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	dealState, _, err := s.GetDealState(ctx, dsn)
	if err != nil {
		return nil, false, err
	}

	state := &State{}
	state.SetDBState(dbState)
	if err := state.SetBlockStates(blockStates); err != nil {
		return nil, false, err
	}

	if err := state.SetBlockMethodStates(blockMethodStates); err != nil {
		return nil, false, err
	}

	if err := state.SetActorStates(actorStates); err != nil {
		return nil, false, err
	}

	if err := state.SetActorMethodStates(actorMethodStates); err != nil {
		return nil, false, err
	}

	if err := state.SetActorTransferStates(actorTransferStates); err != nil {
		return nil, false, err
	}

	if err := state.SetMinedStates(minedStates); err != nil {
		return nil, false, err
	}

	if err := state.SetLargeAmountTransferStates(largeAmountTransferStates); err != nil {
		return nil, false, err
	}

	state.SetDealState(dealState)

	return state, true, nil
}

func (s *Segment) GetDBState(ctx context.Context, dsn string) (*model.DBState, bool, error) {
	cur, err := s.rdb.Find(ctx, "DBState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, false, err
	}

	var res []*model.DBState
	if err = cur.All(ctx, &res); err != nil {
		return nil, false, err
	}

	if len(res) == 0 {
		return nil, false, nil
	}

	return res[0], true, nil
}

// todo: 后面统一获取SegmentState
func (s *Segment) GetBlockStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "BlockState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, nil
	}

	return res, nil
}

func (s *Segment) GetBlockMethodStates(ctx context.Context, dsn string, methodName string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "BlockMethodState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "MethodName", Value: methodName}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetAllBlockMethodStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "BlockMethodState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetActorStates(ctx context.Context, dsn string, actorID string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "ActorID", Value: actorID}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetAllActorStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetActorMethodStates(ctx context.Context, dsn string, actorID string, methodName string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorMethodState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "ActorID", Value: actorID}, {Key: "MethodName", Value: methodName}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetAllActorMethodStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorMethodState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetActorTransferStates(ctx context.Context, dsn string, actorID string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorTransferState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "ActorID", Value: actorID}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetAllActorTransferStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "ActorTransferState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Segment) GetMinedStates(ctx context.Context, dsn string, actorID string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "MinedState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "ActorID", Value: actorID}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

func (s *Segment) GetAllMinedStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "MinedState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

func (s *Segment) GetLargeAmountTransferStates(ctx context.Context, dsn string) ([]model.SegmentState, error) {
	cur, err := s.rdb.Find(ctx, "LargeAmountTransferState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return nil, err
	}

	var res []model.SegmentState
	if err = cur.All(ctx, &res); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

//func (s *Segment) GetAllDealActorStates(ctx context.Context, dsn string) ([]model.SegmentDealState, error) {
//	cur, err := s.rdb.Find(ctx, "DealActorState", bson.D{{Key: "Dsn", Value: dsn}})
//	if err != nil {
//		return nil, err
//	}
//
//	var res []model.SegmentDealState
//	if err = cur.All(ctx, &res); err != nil {
//		return nil, err
//	}
//
//	if len(res) == 0 {
//		return nil, nil
//	}
//
//	return res, nil
//}

//func (s *Segment) GetDealActorStates(ctx context.Context, dsn string, actorID string) ([]model.SegmentDealState, error) {
//	cur, err := s.rdb.Find(ctx, "DealActorState", bson.D{{Key: "Dsn", Value: dsn}, {Key: "ActorID", Value: actorID}})
//	if err != nil {
//		return nil, err
//	}
//
//	var res []model.SegmentDealState
//	if err = cur.All(ctx, &res); err != nil {
//		return nil, err
//	}
//
//	if len(res) == 0 {
//		return nil, nil
//	}
//
//	return res, nil
//}

func (s *Segment) SetBlockState(ctx context.Context, log *zap.SugaredLogger, blockStates []model.SegmentState) error {
	docs := make([]interface{}, 0)
	for _, bs := range blockStates {
		docs = append(docs, bs)
	}

	// todo: 重复插入
	ires, err := s.db.Insert(ctx, "BlockState", docs)
	if err != nil {
		return fmt.Errorf("update blockstate failed: %w", err)
	}

	log.Infof("blockstates inserted: %v/%v", ires, len(blockStates))

	return nil
}

func (s *Segment) GetDealState(ctx context.Context, dsn string) (model.DealState, bool, error) {
	cur, err := s.rdb.Find(ctx, "DealState", bson.D{{Key: "Dsn", Value: dsn}})
	if err != nil {
		return model.DealState{}, false, err
	}

	var res []model.DealState
	if err = cur.All(ctx, &res); err != nil {
		return model.DealState{}, false, err
	}

	if len(res) == 0 {
		return model.DealState{}, false, nil
	}

	return res[0], true, nil
}
