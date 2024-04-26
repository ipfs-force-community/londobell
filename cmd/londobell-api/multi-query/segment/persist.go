package segment

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/lib/limiter"
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

func (a *AddUpState) UpdateDBState(dbState *model.DBState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.dbState = dbState
}

func (a *AddUpState) UpdateBlockStates(blockStates []model.SegmentState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.blockStates = blockStates
}

func (a *AddUpState) UpdateBlockMethodStates(blockMethodStates []model.SegmentState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.blockMethodStates = blockMethodStates
}

type PersistState struct {
	Dsn        string
	Start      int64
	End        int64
	Count      int64
	NextStart  int64
	MethodName string
}

// 累加
func (s *Segment) AddUpDBState(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, state *State, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "DBState")

	dbState := state.GetDBState()
	dbState.EndEpoch = abi.ChainEpoch(nextEndEpoch)
	startEpoch, dsn, interval, dType := int64(dbState.StartEpoch), dbState.Dsn, dbState.Interval, dbState.DType

	starttime := time.Now()
	var err error
	defer func() {
		if err != nil {
			rlog.Infof("addup DBState failed between %v and %v, elapsed: %v, err: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String(), err)
		} else {
			rlog.Infof("addup DBState successfully between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
		}
	}()

	err = s.db.FindOneAndUpdate(ctx, "DBState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dbState.StartEpoch)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: dbState.StartEpoch}, {Key: "DType", Value: dType}, {Key: "Interval", Value: interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: dbState.EndEpoch}}}})
	if err != nil {
		return fmt.Errorf("update dbstate failed: %w", err)
	}

	// refresh cache dbstate
	newDBState := *dbState
	newDBState.EndEpoch = abi.ChainEpoch(nextEndEpoch)
	addUpState.UpdateDBState(&newDBState)

	return nil
}

func (s *Segment) AddUpBlockState(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, state *State, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "BlockState")

	dbState := state.GetDBState()
	startEpoch := int64(dbState.StartEpoch)

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup BlockState done between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
	}()

	blockStates := state.GetBlockStates()
	newBlockStates, err := s.AddUpSegment(ctx, rlog, blockStates, state, nextEndEpoch, cols, "", common.BlockStates)
	if err != nil {
		rlog.Errorf("AddUpSegment failed: %v", err)
		return err
	}

	// refresh cache blockStates
	addUpState.UpdateBlockStates(newBlockStates)

	return nil
}

func (s *Segment) AddUpBlockMethodStates(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, state *State, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "BlockMethodStates")

	dbState := state.GetDBState()
	startEpoch := int64(dbState.StartEpoch)

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup BlockState done between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
	}()

	allNewBlockMethodStates := make([]model.SegmentState, 0)
	for _, method := range util.AllMethodList {
		// todo: 是否并发; 根据方法灵活选择internal

		if method == "Other" {
			method = ""
		}
		blockMethodStates := state.GetBlockMethodStates(method)
		newBlockMethodStates, err := s.AddUpSegment(ctx, rlog, blockMethodStates, state, nextEndEpoch, cols, method, common.BlockMethodStates)
		if err != nil {
			rlog.Errorf("AddUpSegment failed: %v", err)
			return err
		}

		allNewBlockMethodStates = append(allNewBlockMethodStates, newBlockMethodStates...)
	}

	// refresh cache blockMethodStates
	addUpState.UpdateBlockStates(allNewBlockMethodStates)

	return nil
}

func (s *Segment) AddUpSegment(ctx context.Context, log *zap.SugaredLogger, segmentState []model.SegmentState, state *State, nextEndEpoch int64, cols common.Collections, methodName string, segmentType common.SegmentType) ([]model.SegmentState, error) {
	rlog := log.With("AddUp", "Segment")

	dbState := state.GetDBState()
	startEpoch, dsn, interval := int64(dbState.StartEpoch), dbState.Dsn, dbState.Interval

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup segment state done between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
	}()

	sort.Slice(segmentState, func(i, j int) bool {
		return segmentState[i].StartEpoch > segmentState[j].StartEpoch
	})

	var endEpoch int64
	if len(segmentState) == 0 {
		endEpoch = startEpoch
	} else {
		endEpoch = int64(segmentState[0].EndEpoch)
	}

	if nextEndEpoch <= endEpoch {
		rlog.Infof("skip for nextEndEpoch %v <= endEpoch %v", nextEndEpoch, endEpoch)
		return nil, nil
	}

	var (
		todoSegmentStates = make([]PersistState, 0)
		initialBlockState PersistState
		initialEpoch      int64
	)

	newSegmentStates := make([]model.SegmentState, 0)

	length := len(segmentState)
	if length == 0 {
		initialEpoch = startEpoch
		initialBlockState = PersistState{Dsn: dsn, Start: startEpoch, NextStart: startEpoch, MethodName: methodName}
	} else {
		start, end, count := int64(segmentState[0].StartEpoch), int64(segmentState[0].EndEpoch), segmentState[0].Count
		if end-start == interval {
			// new next blockState
			initialEpoch = end
			initialBlockState = PersistState{Dsn: dsn, Start: end, NextStart: end, MethodName: methodName}
			newSegmentStates = segmentState[:]
		} else {
			initialEpoch = start
			initialBlockState = PersistState{Dsn: dsn, Start: start, Count: count, NextStart: end, MethodName: methodName}
			newSegmentStates = segmentState[1:]
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
			todoSegmentStates = append(todoSegmentStates, initialBlockState)
			first = false
		} else {
			todoSegmentStates = append(todoSegmentStates, PersistState{Dsn: dsn, Start: initialEpoch, End: end, NextStart: initialEpoch, MethodName: methodName})
		}

		initialEpoch = end
	}

	lim := limiter.New(s.opts.BatchInsertLimit)
	var lk sync.Mutex

	var ewg multierror.Group
	for _, todoSegmentState := range todoSegmentStates {
		todoSegmentState := todoSegmentState
		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				return nil
			}

			defer func() {
				lim.Release(ctx)
			}()

			starttime := time.Now()
			start, end, dsn, nextStart, methodName := todoSegmentState.Start, todoSegmentState.End, todoSegmentState.Dsn, todoSegmentState.NextStart, todoSegmentState.MethodName
			defer func() {
				rlog.Infof("addup segment successfully between %v and %v, count: %v, elapsed: %v", start, end, todoSegmentState.Count, time.Now().Sub(starttime).String())
			}()

			for _, col := range cols.Cols {
				if col != nil && col.Name() == "ExecTrace" {
					switch segmentType {
					case common.BlockStates:
						blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: nextStart}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}}, {Key: "IsBlock", Value: true}}
						count, err := col.CountDocuments(ctx, blockFilter)
						if err != nil {
							return fmt.Errorf("count for block messages failed: %w", err)
						}

						todoSegmentState.Count += count

						err = s.db.FindOneAndUpdate(ctx, "BlockState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, start)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: start}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: end}, {Key: "Count", Value: todoSegmentState.Count}}}})
						if err != nil {
							return fmt.Errorf("update blockstate failed: %w", err)
						}

						lk.Lock()
						newSegmentStates = append(newSegmentStates, model.SegmentState{ID: fmt.Sprintf("%v-%v", dsn, start), Dsn: dsn, StartEpoch: abi.ChainEpoch(start), EndEpoch: abi.ChainEpoch(end), Count: todoSegmentState.Count})
						lk.Unlock()
					case common.BlockMethodStates:
						blockMethodFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: nextStart}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}}, {Key: "IsBlock", Value: true}, {Key: "Msg.MethodName", Value: methodName}}
						count, err := col.CountDocuments(ctx, blockMethodFilter)
						if err != nil {
							return fmt.Errorf("count for blockmethod messages failed: %w", err)
						}

						todoSegmentState.Count += count

						// todo: "" method的_id是否正常
						err = s.db.FindOneAndUpdate(ctx, "BlockMethodState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v-%v", dsn, start, methodName)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: start}, {Key: "MethodName", Value: methodName}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: end}, {Key: "Count", Value: todoSegmentState.Count}}}})
						if err != nil {
							return fmt.Errorf("update blockstate failed: %w", err)
						}

						lk.Lock()
						newSegmentStates = append(newSegmentStates, model.SegmentState{ID: fmt.Sprintf("%v-%v-%v", dsn, start, methodName), Dsn: dsn, StartEpoch: abi.ChainEpoch(start), EndEpoch: abi.ChainEpoch(end), Count: todoSegmentState.Count, MethodName: methodName})
						lk.Unlock()
					}

					return nil
				}
			}

			return fmt.Errorf("no ExecTrace collections")
		})
	}

	if err := ewg.Wait(); err != nil {
		rlog.Infof("addup segment state failed between %v and %v, elapsed: %v, err: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String(), err)
		return nil, err
	}

	return newSegmentStates, nil
}

func (s *Segment) DeleteDBState(ctx context.Context, log *zap.SugaredLogger, dsn string) error {
	filter := bson.D{{Key: "Dsn", Value: dsn}}
	deleteCount, err := s.db.Delete(ctx, "DBState", filter)
	if err != nil {
		log.Errorf("delete DBState for %v failed: %v", dsn, err)
		return err
	}

	log.Infof("delete DBState for %v successfully, deleteCount: %v", dsn, deleteCount)

	return nil
}

func (s *Segment) DeleteBlockState(ctx context.Context, log *zap.SugaredLogger, dsn string) error {
	filter := bson.D{{Key: "Dsn", Value: dsn}}
	deleteCount, err := s.db.Delete(ctx, "BlockState", filter)
	if err != nil {
		log.Errorf("delete BlockState for %v failed: %v", dsn, err)
		return err
	}

	log.Infof("delete BlockState for %v successfully, deleteCount: %v", dsn, deleteCount)

	return nil
}

func (s *Segment) DeleteBlockMethodState(ctx context.Context, log *zap.SugaredLogger, dsn string) error {
	filter := bson.D{{Key: "Dsn", Value: dsn}}
	deleteCount, err := s.db.Delete(ctx, "BlockMethodState", filter)
	if err != nil {
		log.Errorf("delete BlockMethodState for %v failed: %v", dsn, err)
		return err
	}

	log.Infof("delete BlockMethodState for %v successfully, deleteCount: %v", dsn, deleteCount)

	return nil
}

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

	//dealState, _, err := s.GetDealState(ctx, dsn)
	//if err != nil {
	//	return nil, false, err
	//}

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

	//state.SetDealState(dealState)

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
