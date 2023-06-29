package segment

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
)

// 累加
func (s *Segment) AddUpBlockState(ctx context.Context, log *zap.SugaredLogger, nextEndEpoch abi.ChainEpoch, state *State, cols common.Collections) error {
	rlog := log.With("AddUp", "BlockState")

	dbState := state.GetDBState()
	blockStates := state.GetBlockStates()

	newBlockStates := make([]model.SegmentState, 0)

	startEpoch, dsn, interval := dbState.StartEpoch, dbState.Dsn, dbState.Interval

	starttime := time.Now()
	defer func() {
		rlog.Infof("addup BlockState successfully between %v and %v, elapsed: %v", startEpoch, nextEndEpoch, time.Now().Sub(starttime).String())
	}()

	sort.Slice(blockStates, func(i, j int) bool {
		return blockStates[i].StartEpoch > blockStates[j].StartEpoch
	})

	var endEpoch abi.ChainEpoch
	if len(blockStates) == 0 {
		endEpoch = startEpoch
	} else {
		endEpoch = blockStates[0].EndEpoch
	}

	if nextEndEpoch <= endEpoch {
		rlog.Infof("skip for nextEndEpoch %v <= endEpoch %v", nextEndEpoch, endEpoch)
		return nil
	}

	var (
		todoBlockStates   = make([]model.SegmentState, 0)
		initialBlockState model.SegmentState
		initialEpoch      abi.ChainEpoch
	)

	length := len(blockStates)
	if length == 0 {
		initialEpoch = startEpoch
		initialBlockState = model.SegmentState{Dsn: dsn, StartEpoch: startEpoch}
	} else {
		start, end, count := blockStates[0].StartEpoch, blockStates[0].EndEpoch, blockStates[0].Count
		if end-start == interval {
			// new next blockState
			initialEpoch = end
			initialBlockState = model.SegmentState{Dsn: dsn, StartEpoch: end, Count: count}
			newBlockStates = blockStates[:]
		} else {
			initialEpoch = start
			initialBlockState = model.SegmentState{Dsn: dsn, StartEpoch: start, Count: count}
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
			initialBlockState.EndEpoch = end
			todoBlockStates = append(todoBlockStates, initialBlockState)
			first = false
		} else {
			todoBlockStates = append(todoBlockStates, model.SegmentState{Dsn: dsn, StartEpoch: initialEpoch, EndEpoch: end})
		}

		initialEpoch = end
	}

	newBlockStates = append(newBlockStates, todoBlockStates...)
	lim := limiter.New(s.opts.BatchInsertLimit)

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
			start, end, dsn := todoBlockState.StartEpoch, todoBlockState.EndEpoch, todoBlockState.Dsn
			defer func() {
				rlog.Infof("addup BlockMsgsCount successfully between %v and %v, count: %v, elapsed: %v", start, end, todoBlockState.Count, time.Now().Sub(starttime).String())
			}()

			blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: start}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: end}}}, {Key: "IsBlock", Value: true}}

			for _, col := range cols.Cols {
				if col != nil && col.Name() == "ExecTrace" {
					count, err := col.CountDocuments(ctx, blockFilter)
					if err != nil {
						return fmt.Errorf("count for block messages failed: %w", err)
					}

					todoBlockState.Count += count

					// todo: 测试插入成功
					err = s.db.FindOneAndUpdate(ctx, "BlockState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, start)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: start}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: todoBlockState.EndEpoch}, {Key: "Count", Value: todoBlockState.Count}}}})
					if err != nil {
						return fmt.Errorf("update blockstate failed: %w", err)
					}

					return nil
				}
			}

			return fmt.Errorf("no ExecTrace collections")
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	dbState.EndEpoch = nextEndEpoch
	err := s.db.FindOneAndUpdate(ctx, "DBState", bson.D{{Key: "_id", Value: fmt.Sprintf("%v-%v", dsn, dbState.StartEpoch)}, {Key: "Dsn", Value: dsn}, {Key: "StartEpoch", Value: dbState.StartEpoch}, {Key: "DType", Value: dbState.DType}, {Key: "Interval", Value: dbState.Interval}}, bson.D{{Key: "$set", Value: bson.D{{Key: "EndEpoch", Value: dbState.EndEpoch}}}})
	if err != nil {
		return fmt.Errorf("update dbstate failed: %w", err)
	}

	if err := state.SetBlockStates(newBlockStates); err != nil {
		return err
	}

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
