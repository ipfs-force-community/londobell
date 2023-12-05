package segment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (a *AddUpState) UpdateActorStates(actorStates []model.SegmentState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.actorStates = actorStates
}

func (a *AddUpState) UpdateBlockMethodStates(blockMethodStates []model.SegmentState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.blockMethodStates = blockMethodStates
}

func (a *AddUpState) UpdateActorMethodStates(actorMethodStates []model.SegmentState) {
	a.lk.Lock()
	defer a.lk.Unlock()

	a.State.actorMethodStates = actorMethodStates
}

type PersistState struct {
	Dsn        string
	Start      int64
	End        int64
	Count      int64
	MethodName string
	ActorID    string
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
			rlog.Infof("failed between %d and %d, elapsed: %s, err: %v", startEpoch, nextEndEpoch, time.Since(starttime), err)
		} else {
			rlog.Infof("successfully between %d and %d, elapsed: %s", startEpoch, nextEndEpoch, time.Since(starttime))
		}
	}()

	err = s.db.FindOneAndUpdate(ctx, "DBState", bson.D{
		{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dbState.StartEpoch)},
		{Key: "StartEpoch", Value: dbState.StartEpoch},
		{Key: "Dsn", Value: dsn},
		{Key: "DType", Value: dType},
		{Key: "Interval", Value: interval},
	},
		bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: dbState.EndEpoch}}}})
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
		rlog.Infof("done between %d and %d, elapsed: %s", startEpoch, nextEndEpoch, time.Since(starttime))
	}()

	blockStates := state.GetBlockStates()
	newBlockStates, err := s.AddUpSegment(ctx, rlog, blockStates, state, nextEndEpoch, cols, "", "", common.BlockStates)
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
		rlog.Infof("done between %d and %d, elapsed: %s", startEpoch, nextEndEpoch, time.Since(starttime))
	}()

	allNewBlockMethodStates := make([]model.SegmentState, 0)
	for _, method := range util.AllMethodList {
		// todo: 是否并发; 根据方法灵活选择internal

		if method == "Other" {
			method = ""
		}
		blockMethodStates := state.GetBlockMethodStates(method)
		newBlockMethodStates, err := s.AddUpSegment(ctx, rlog, blockMethodStates, state, nextEndEpoch, cols, method, "", common.BlockMethodStates)
		if err != nil {
			rlog.Errorf("AddUpSegment failed: %v", err)
			return err
		}

		allNewBlockMethodStates = append(allNewBlockMethodStates, newBlockMethodStates...)
	}

	// refresh cache blockMethodStates
	addUpState.UpdateBlockMethodStates(allNewBlockMethodStates)

	return nil
}

func (s *Segment) AddUpActorStates(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, state *State, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "ActorStates")

	dbState := state.GetDBState()
	startEpoch := int64(dbState.StartEpoch)

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup done between %d and %d, elapsed: %s", startEpoch, nextEndEpoch, time.Since(starttime))
	}()

	actorStates := state.GetAllActorStates()
	newActorStates, err := s.AddUpSegment(ctx, rlog, actorStates, state, nextEndEpoch, cols, "", "", common.ActorStates)
	if err != nil {
		rlog.Errorf("AddUpSegment failed: %v", err)
		return err
	}
	// refresh cache blockMethodStates

	addUpState.UpdateActorStates(append(actorStates, newActorStates...))

	return nil
}

func (s *Segment) AddUpActorMethodStates(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch int64, state *State, addUpState *AddUpState, cols common.Collections) error {
	rlog := log.With("AddUp", "ActorMethodStates")

	dbState := state.GetDBState()
	startEpoch := int64(dbState.StartEpoch)

	starttime := time.Now()
	defer func() {
		rlog.Infof("done between %d and %d, elapsed: %s", startEpoch, nextEndEpoch, time.Since(starttime))
	}()

	actorStates := state.GetAllActorStates()
	newActorStates, err := s.AddUpSegment(ctx, rlog, actorStates, state, nextEndEpoch, cols, "", "", common.ActorMethodStates)
	if err != nil {
		rlog.Errorf("AddUpSegment failed: %v", err)
		return err
	}

	addUpState.UpdateActorMethodStates(append(actorStates, newActorStates...))

	return nil
}

func (s *Segment) AddUpSegment(ctx context.Context, log *zap.SugaredLogger, segmentState []model.SegmentState, state *State, nextEndEpoch int64, cols common.Collections, methodName, actorID string, segmentType common.SegmentType) ([]model.SegmentState, error) {
	rlog := log.With("AddUp", "Segment")

	dbState := state.GetDBState()
	dbStartEpoch, dsn, interval := int64(dbState.StartEpoch), dbState.Dsn, dbState.Interval

	starttime := time.Now()
	stateCol := getSegmentColName(segmentType)
	if stateCol == "" {
		return nil, fmt.Errorf("not support segment type: %v", segmentType)
	}
	defer func() {
		rlog.Infof("%s done: %d-%d, spent: %s", stateCol, dbStartEpoch, nextEndEpoch, time.Since(starttime))
	}()

	sort.Slice(segmentState, func(i, j int) bool {
		return segmentState[i].StartEpoch > segmentState[j].StartEpoch
	})

	var (
		todoSegmentStates = make([]PersistState, 0)
		stateEndEpoch     int64
		newSegmentStates  = make([]model.SegmentState, 0)
	)

	if len(segmentState) == 0 {
		stateEndEpoch = dbStartEpoch
	} else {
		stateEndEpoch = int64(segmentState[0].EndEpoch)
	}

	if nextEndEpoch <= stateEndEpoch {
		rlog.Infof("skip,want state epoch :%d,have max epoch %d", nextEndEpoch, stateEndEpoch)
		return nil, nil
	}

	col := getSegCol(cols.Cols, segmentType)
	totalCount, err := getTotalCount(ctx, col, stateEndEpoch, nextEndEpoch, segmentType, methodName)
	if err != nil {
		return nil, err
	}
	log.Infof("getTotalCount done,total: %d,method: %s", totalCount, methodName)
	for stateEndEpoch < nextEndEpoch {
		// TODO 不同的分段类型应有不同的interval,如果没必要分别控制,可合并同表的segment
		end := stateEndEpoch + interval
		if end > nextEndEpoch {
			end = nextEndEpoch
		}
		todoSegmentStates = append(todoSegmentStates, PersistState{Dsn: dsn, Start: stateEndEpoch,
			End: end, MethodName: methodName})

		stateEndEpoch = end
	}

	lim := limiter.New(s.opts.BatchInsertLimit)
	var (
		lk  sync.Mutex
		ewg multierror.Group
	)

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
			start, end, dsn, methodName := todoSegmentState.Start, todoSegmentState.End, todoSegmentState.Dsn, todoSegmentState.MethodName
			defer func() {
				rlog.Infof("success: %d-%d, count: %d, spent: %s", start, end, todoSegmentState.Count, time.Since(starttime))
			}()

			switch segmentType {
			case common.BlockStates:
				blockFilter := bson.D{
					{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}},
					{Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}},
					{Key: "IsBlock", Value: true},
				}
				count, err := col.CountDocuments(ctx, blockFilter)
				if err != nil {
					return fmt.Errorf("count for block messages failed: %w", err)
				}

				todoSegmentState.Count += count
				lk.Lock()
				newSegmentStates = append(newSegmentStates, model.SegmentState{ID: fmt.Sprintf("%v", start), Dsn: dsn, StartEpoch: abi.ChainEpoch(start), EndEpoch: abi.ChainEpoch(end), Count: todoSegmentState.Count})
				lk.Unlock()
			case common.BlockMethodStates:
				blockMethodFilter := bson.D{
					{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}},
					{Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}},
					{Key: "IsBlock", Value: true},
					{Key: "Msg.MethodName", Value: methodName},
				}
				count, err := col.CountDocuments(ctx, blockMethodFilter)
				if err != nil {
					return fmt.Errorf("count for blockmethod messages failed: %w", err)
				}

				todoSegmentState.Count += count

				lk.Lock()
				newSegmentStates = append(newSegmentStates, model.SegmentState{ID: fmt.Sprintf("%v-%v", start, methodName), Dsn: dsn, StartEpoch: abi.ChainEpoch(start), EndEpoch: abi.ChainEpoch(end), Count: todoSegmentState.Count, MethodName: methodName})
				lk.Unlock()
			case common.ActorStates, common.ActorMethodStates:
				var (
					actors []struct {
						ActorID    string `bson:"ActorID"`
						MethodName string `bson:"MethodName"`
					}
					segments      []model.SegmentState
					newDataFilter = bson.D{
						{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}},
						{Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}},
						{Key: "IsBlock", Value: true},
					}
				)
				cusor, err := col.Find(ctx, newDataFilter, options.Find().SetProjection(bson.M{"ActorID": 1, "MethodName": 1}))
				if err != nil {
					return fmt.Errorf("find actor message failed: %w", err)
				}
				err = cusor.All(ctx, &actors)
				if err != nil {
					return fmt.Errorf("get actor message failed: %w", err)
				}
				todoSegmentState.Count = int64(len(actors))
				actorMap := make(map[string]int64)
				actorMethodMap := make(map[string]int64)

				for _, actor := range actors {
					actorMap[actor.ActorID]++
					actorMethodMap[fmt.Sprintf("%v-%v", actor.ActorID, actor.MethodName)]++
				}
				if segmentType == common.ActorStates {
					for actorID, count := range actorMap {
						segment := model.SegmentState{
							ID:  fmt.Sprintf("%v-%v", start, actorID),
							Dsn: dsn, StartEpoch: abi.ChainEpoch(start),
							EndEpoch: abi.ChainEpoch(end), Count: count,
							ActorID: actorID}
						segments = append(segments, segment)

					}
					lk.Lock()
					newSegmentStates = append(newSegmentStates, segments...)
					lk.Unlock()
				} else {

					extractActorMethod := func(actorMethod string) (actorID string, methodName string) {
						actorID = actorMethod[:strings.Index(actorMethod, "-")]
						methodName = actorMethod[strings.Index(actorMethod, "-")+1:]
						return
					}
					for actorMethod, count := range actorMethodMap {
						actorID, methodName := extractActorMethod(actorMethod)
						segment := model.SegmentState{
							ID:  fmt.Sprintf("%v-%v-%v", start, actorID, methodName),
							Dsn: dsn, StartEpoch: abi.ChainEpoch(start),
							EndEpoch: abi.ChainEpoch(end), Count: count,
							MethodName: methodName, ActorID: actorID}
						segments = append(segments, segment)
					}

					lk.Lock()
					newSegmentStates = append(newSegmentStates, segments...)
					lk.Unlock()

				}
			}
			return nil
		})
	}
	if err := ewg.Wait(); err != nil {
		rlog.Infof("addup segment state failed between %d and %d, elapsed: %s, err: %v", dbStartEpoch, nextEndEpoch, time.Since(starttime), err)
		return nil, err
	}
	newSegmentStates, segCount := compactSegment(newSegmentStates, segmentType)
	log.Infof("compactSegment done,total: %d", segCount)
	if segCount != totalCount {
		err = fmt.Errorf("segment count: %d not equal total count : %d", segCount, totalCount)
		log.Error(err)
		return nil, err
	}
	var docs []interface{}
	if len(newSegmentStates) > 0 {
		for _, seg := range newSegmentStates {
			docs = append(docs, seg)
		}
		// TODO 应当全局校验连接,但变动较大且其他地方暂未出现超时情况,暂不处理
		err = s.wcli.Ping(ctx, nil)
		if err != nil {
			log.Warn("mongo disconnect ReConnect...")
			err = s.ReConnect(ctx)
			if err != nil {
				log.Error("failed,", err)
				return nil, err
			}
		}
		// 每次插入10000避免socket超时
		for start, end := 0, len(docs); start < end; start += 10000 {
			tempEnd := start + 10000
			if tempEnd > end {
				tempEnd = end
			}
			inserted, err := s.db.Insert(ctx, stateCol, docs[start:tempEnd])
			if err != nil {
				err = fmt.Errorf("update %s failed: %w", stateCol, err)
				log.Error(err)
				return nil, err
			}
			log.Infof("%s inserted %d docs", stateCol, inserted)
		}

	}

	return newSegmentStates, nil
}

func getSegCol(cols []*mongo.Collection, sType common.SegmentType) *mongo.Collection {
	var colName string
	if sType == common.ActorStates || sType == common.ActorMethodStates {
		colName = "ActorMessage"
	}

	switch sType {
	case common.ActorStates, common.ActorMethodStates:
		colName = "ActorMessage"
	case common.BlockStates, common.BlockMethodStates:
		colName = "ExecTrace"
	default:
		// should not happen
		colName = ""
	}
	for _, col := range cols {
		if col != nil && col.Name() == colName {
			return col
		}
	}
	return nil
}

func getTotalCount(ctx context.Context, col *mongo.Collection, start, end int64, segmentType common.SegmentType, method string) (int64, error) {
	filter := bson.D{
		{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}},
		{Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}},
		{Key: "IsBlock", Value: true},
	}
	if segmentType == common.BlockMethodStates {
		filter = bson.D{
			{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}},
			{Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}},
			{Key: "Msg.MethodName", Value: method},
			{Key: "IsBlock", Value: true},
		}
	}
	return col.CountDocuments(ctx, filter)

}

func getSegmentColName(sType common.SegmentType) string {
	switch sType {
	case common.ActorStates:
		return "ActorState"
	case common.ActorMethodStates:
		return "ActorMethodState"
	case common.BlockStates:
		return "BlockState"
	case common.BlockMethodStates:
		return "BlockMethodState"
	default:
		// should not happen
		return ""
	}

}

func compactSegment(segments []model.SegmentState, sType common.SegmentType) ([]model.SegmentState, int64) {
	var (
		// 分段信息,用于限制range count大小
		spanCount = make(map[string]int64)
		// 分段起始Epoch
		spanEStart  = make(map[string]abi.ChainEpoch)
		res         []model.SegmentState
		countRange  = int64(10000)
		objSegments = make(map[string][]model.SegmentState)
		totalCount  int64
	)
	for _, seg := range segments {
		// segObj := getSegObj(seg, sType)
		var segObj string
		switch sType {
		case common.ActorStates:
			segObj = seg.ActorID
		case common.ActorMethodStates:
			segObj = seg.ActorID + seg.MethodName
		case common.BlockStates:
			segObj = "block"
		case common.BlockMethodStates:
			segObj = seg.MethodName
		default:
			// should not happen
			segObj = ""
		}
		objSegments[segObj] = append(objSegments[segObj], seg)
	}

	for segObj, segs := range objSegments {
		sort.Slice(segs, func(i, j int) bool {
			return segs[i].StartEpoch < segs[j].StartEpoch
		})
		for _, seg := range segs {
			if spanEStart[segObj] == 0 {
				spanEStart[segObj] = segs[0].StartEpoch
			}
			// 过滤空数据
			if spanCount[segObj] == 0 {
				spanEStart[segObj] = seg.StartEpoch
			}
			spanCount[segObj] += seg.Count
			if spanCount[segObj] >= countRange || seg.EndEpoch == segs[len(segs)-1].EndEpoch {
				spanEEnd := seg.EndEpoch
				span := model.SegmentState{
					StartEpoch: spanEStart[segObj],
					EndEpoch:   spanEEnd,
					Count:      spanCount[segObj],
					Dsn:        seg.Dsn,
					ActorID:    seg.ActorID,
					MethodName: seg.MethodName,
				}

				switch sType {
				case common.ActorStates:
					span.ID = fmt.Sprintf("%v-%v", span.StartEpoch, span.ActorID)
				case common.ActorMethodStates:
					span.ID = fmt.Sprintf("%v-%v-%v", span.StartEpoch, span.ActorID, span.MethodName)
				case common.BlockStates:
					span.ID = fmt.Sprintf("%v", span.StartEpoch)
				case common.BlockMethodStates:
					span.ID = fmt.Sprintf("%v-%v", span.StartEpoch, span.MethodName)
				default:
					// should not happen
					span.ID = ""
				}

				res = append(res, span)
				totalCount += span.Count
				spanCount[segObj] = 0
				// spanEStart[segObj] = spanEEnd
			}
		}

	}
	return res, totalCount
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

func (s *Segment) DeleteActorState(ctx context.Context, log *zap.SugaredLogger, dsn string) error {
	filter := bson.D{{Key: "Dsn", Value: dsn}}
	deleteCount, err := s.db.Delete(ctx, "ActorState", filter)
	if err != nil {
		log.Errorf("delete ActorState for %v failed: %v", dsn, err)
		return err
	}

	log.Infof("delete ActorState for %v successfully, deleteCount: %v", dsn, deleteCount)

	return nil
}

func (s *Segment) DeleteActorMethodState(ctx context.Context, log *zap.SugaredLogger, dsn string) error {
	filter := bson.D{{Key: "Dsn", Value: dsn}}
	deleteCount, err := s.db.Delete(ctx, "ActorMethodState", filter)
	if err != nil {
		log.Errorf("delete ActorMethodState for %v failed: %v", dsn, err)
		return err
	}

	log.Infof("delete ActorMethodState for %v successfully, deleteCount: %v", dsn, deleteCount)

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
