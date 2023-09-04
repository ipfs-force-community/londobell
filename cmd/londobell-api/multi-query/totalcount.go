package multiquery

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hashicorp/go-multierror"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

var (
	log = logging.Logger("multi-query")
)

//todo:test
var (
	IncrementalEndEpoch = abi.ChainEpoch(1960619)
)

var (
	addupOnce sync.Once
	addupes   = make([]func(ctx context.Context, log *zap.SugaredLogger, state *segment.State, addUpState *segment.AddUpState, cols common.Collections, seg *segment.Segment, nextEndEpoch int64) error, 0)

	AllMethods     = make(map[string]int64, 0)
	alk            sync.RWMutex
	lastUpdateTime time.Time
)

func init() {
	addupOnce.Do(func() {
		addupes = append(addupes, AddUpBlockState, AddUpBlockMethodStates /* AddUpDealActorState */)
	})
}

func PeriodicRefreshDataBaseState(ctx context.Context, log *zap.SugaredLogger, dbsm *DataBaseStateManager) {
	tick := time.NewTicker(15 * time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			start := time.Now()
			log.Info("begin PeriodicRefreshDataBaseState for formal")
			if err := RefreshFormalDataBaseState(ctx, log, dbsm); err != nil {
				log.Error(err)
				continue
			}

			log.Infof("finish PeriodicRefreshDataBaseState for formal, elapsed: %v", time.Now().Sub(start).String())
		}
	}
}

func GetFinalHeightForFormalDB(ctx context.Context, dbsm *DataBaseStateManager) (abi.ChainEpoch, error) {
	cols, ok := dbsm.GetDBCollections(dbsm.GetFormalCfg().Url())
	if !ok {
		return 0, fmt.Errorf("url %v not found in DBCollectionsMap", dbsm.GetFormalCfg().Url())
	}

	return GetFinalHeight(ctx, cols)
}

//func TestPeriodicRefreshDataBaseState(ctx context.Context, dbsm *DataBaseStateManager) {
//	tick := time.NewTicker(10 * time.Minute) //todo:test
//	defer tick.Stop()
//
//	for {
//		select {
//		case <-tick.C:
//			//res, err := GetFinalHeightForFormalDB(ctx, dbsm)
//			//if err != nil {
//			//	log.Error(err)
//			//	continue
//			//}
//			//
//			//if len(res) == 0 { // todo
//			//	log.Warnf("no data in FinalHeight")
//			//	continue
//			//}
//
//			//todo:test
//			if err := TestRefreshFormalDataBaseState(ctx, dbsm, IncrementalEndEpoch); err != nil {
//				log.Error(err)
//				continue
//			}
//			IncrementalEndEpoch += 60
//		}
//	}
//}

// 只有formal需要定期刷
// todo:会发生多个进程同时操作dbsm.Stm的操作，导致dbState紊乱有错吗？
func RefreshFormalDataBaseState(ctx context.Context, log *zap.SugaredLogger, dbsm *DataBaseStateManager) error {
	formal := dbsm.GetFormalCfg()

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
		return nil
	}

	dsn := formal.Url()
	state, found, err := dbsm.GetState(ctx, dsn)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("state of %v not found", dsn)
	}

	cols, ok := dbsm.GetDBCollections(formal.Url())
	if !ok {
		return fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
	}

	newState := *state // todo: 指针get出来是否会导致原来错误
	addUpState := segment.NewAddUpState(newState)

	finalHeight, err := GetFinalHeight(ctx, cols)
	if err != nil {
		return err
	}
	nextEndEpoch := int64(finalHeight + 1)

	var ewg multierror.Group
	// only persist segment, don't change state
	for i := range addupes {
		i := i
		addup := addupes[i]
		ewg.Go(func() error {
			if err := addup(ctx, log, state, addUpState, cols, dbsm.Segment, nextEndEpoch); err != nil {
				return err
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		log.Errorf("RefreshFormalDataBaseState failed: %v", err)
		return err
	}

	// persist dbstate
	if err := AddUpDBState(ctx, log, state, addUpState, cols, dbsm.Segment, nextEndEpoch); err != nil {
		log.Errorf("AddUpDBState faild for nextEndEpoch: %v", nextEndEpoch)
		return err
	}

	// refresh cache
	newState = addUpState.GetState()
	dbsm.DBStateCache.SetState(dsn, &newState)

	log.Infof("RefreshFormalDataBaseState successfully, dbState.EndEpoch: %v", newState.GetEndEpoch())

	return nil
}

func AddUpDBState(ctx context.Context, log *zap.SugaredLogger, state *segment.State, addUpState *segment.AddUpState, cols common.Collections, seg *segment.Segment, nextEndEpoch int64) error {
	return seg.AddUpDBState(ctx, log, nextEndEpoch, state, addUpState, cols)
}

func AddUpBlockState(ctx context.Context, log *zap.SugaredLogger, state *segment.State, addUpState *segment.AddUpState, cols common.Collections, seg *segment.Segment, nextEndEpoch int64) error {
	return seg.AddUpBlockState(ctx, log, nextEndEpoch, state, addUpState, cols)
}

func AddUpBlockMethodStates(ctx context.Context, log *zap.SugaredLogger, state *segment.State, addUpState *segment.AddUpState, cols common.Collections, seg *segment.Segment, nextEndEpoch int64) error {
	return seg.AddUpBlockMethodStates(ctx, log, nextEndEpoch, state, addUpState, cols)
}

//func AddUpDealActorState(ctx context.Context, log *zap.SugaredLogger, addUpState *segment.AddUpState, cols common.Collections, seg *segment.Segment) error {
//	state := addUpState.GetState()
//
//	endDealID, err := GetEndDealID(ctx, cols, state.GetDealStartID())
//	if err != nil {
//		return err
//	}
//
//	nextDealID := int64(endDealID + 1)
//	return seg.AddUpDealActorState(ctx, log, nextDealID, addUpState, cols)
//}

//func TestRefreshFormalDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, finalHeight abi.ChainEpoch) error {
//	start := time.Now() //todo:test
//
//	formal := dbsm.GetFormalCfg()
//
//	if formal.IsInvalidDB() {
//		log.Warnf("db %v is invalid", formal)
//		return nil
//	}
//
//	dbState, found, err := dbsm.Stm.LoadDataBaseState(formal.Url())
//	if err != nil {
//		return err
//	}
//
//	if !found {
//		////todo
//		//dbState, err := InitDataBaseState(ctx, db, true)
//		//if err != nil {
//		//	return 0, nil, err
//		//}
//		//
//		//fmt.Println("InitDataBaseState", dbState)
//		//if err := m.SetDataBaseState(db.Url, *dbState); err != nil {
//		//	return 0, nil, err
//		//}
//		return fmt.Errorf("db %v not found", formal)
//	}
//
//	cols, ok := dbsm.GetDBCollections(formal.Url())
//	if !ok {
//		return fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
//	}
//
//	dbState.EndEpoch = finalHeight + 1
//
//	if err := RefreshBlockMsgs(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshBlockMsgsByMethodName(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorMsgsByMethodName(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorMsgs(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorTransferMsgs(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshMinedMsgsMaps(ctx, &dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshTransfersForLargeAmount(ctx, &dbState, cols); err != nil {
//		return err
//	}
//
//	if err := dbsm.Stm.SetDataBaseState(formal.Url(), dbState); err != nil {
//		return err
//	}
//
//	//todo:test
//	file, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/bell.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
//	if err != nil {
//		log.Errorf("open bell.txt failed: %v", err)
//	}
//	defer file.Close()
//	_, err = io.WriteString(file, fmt.Sprintf("curtime: %v, startEpoch: %v, endEpoch: %v, NextEpochForBlockMsgsCount: %v, BlockMsgsCount: %v, elapsed: %v\n", fmt.Sprintf("%02d%02d%02d", time.Now().Day(), time.Now().Hour(), time.Now().Minute()), dbState.StartEpoch, dbState.EndEpoch, dbState.NextEpochForBlockMsgsCount, dbState.BlockMsgsCount, time.Now().Sub(start).String()))
//	if err != nil {
//		log.Errorf("write bell.txt failed: %v", err)
//	}
//
//	log.Infow("write to bell.txt successfully", "NextEpochForBlockMsgsCount", dbState.NextEpochForBlockMsgsCount)
//
//	return nil
//}

type ConcurrentCountUtils struct {
	CountUtils []CountUtil
	lk         sync.Mutex
}

func refresh(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch, actorID, methodName string, f func(context.Context, *segment.State, common.Collections, *ConcurrentCountUtils, *abi.ChainEpoch, abi.ChainEpoch, string, string) error) ([]CountUtil, error) {
	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	dbs := make([]common.DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	var (
		ewg                  multierror.Group
		concurrentCountUtils = &ConcurrentCountUtils{
			CountUtils: make([]CountUtil, 0),
		}
	)

	for _, db := range dbs {
		db := db
		ewg.Go(func() error {
			if db.IsInvalidDB() {
				return nil
			}

			state, found, err := dbsm.GetState(ctx, db.Url())
			if err != nil {
				return err
			}

			if !found {
				return fmt.Errorf("state of dsn %v not found", db.Url())
			}

			cols, ok := dbsm.GetDBCollections(db.Url())
			if !ok {
				return fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
			}

			err = f(ctx, state, cols, concurrentCountUtils, &tmpStartEpoch, curEpoch, actorID, methodName)
			if err != nil {
				return err
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpState := segment.DefaultState(tmp.Url(), smodel.Tmp, smodel.DefaultInterval, 0, 0, 0, 0)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		err := f(ctx, tmpState, tmpCols, concurrentCountUtils, &tmpStartEpoch, curEpoch, actorID, methodName)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(concurrentCountUtils.CountUtils, func(i, j int) bool {
		return concurrentCountUtils.CountUtils[i].End > concurrentCountUtils.CountUtils[j].End
	})

	return concurrentCountUtils.CountUtils, nil
}

func GetEpochRange(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshEpochRange)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForTipSets(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshTotalCountForTipSets)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetDealRange(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshDealRange)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

//func GetMinerSectorRange(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
//	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshMinerSectorRange)
//	if err != nil {
//		return nil, err
//	}
//
//	return countUtils, nil
//}

func GetTotalCountForActorDeals(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorDeals)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForBlockMsgs(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshTotalCountForBlockMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// methodname: 块消息数
// count_of_blockmessages_by_methodname.js
func GetTotalCountForBlockMsgsByMethodName(ctx context.Context, methodName string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForBlockMsgsByMethodName")

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", methodName, refreshTotalCountForBlockMsgsByMethodName)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// count_of_actormessages_by_methodname.js
func GetTotalCountForActorMsgByMethodName(ctx context.Context, actorID, methodName string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgByMethodName")

	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, methodName, refreshTotalCountForActorMsgByMethodName)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorMsgByMethodName2(ctx context.Context, actor, methodName string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {

	api := fullnode.API.GetAppropriateAPI()

	addr, err := address.NewFromString(buildnet.NetPrefix + actor)
	if err != nil {
		return nil, err
	}

	actorID := actor
	switch addr.Protocol() {
	case address.ID:
		actorID = addr.String()[1:]
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		actorID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for actor %v", actor)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, methodName, refreshTotalCountForActorMsgByMethodName)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

//actorID对应的块消息条数
//all_actors_for_blockmessage.js
//{from, to}, 出现即消息数+1，from和to为同一地址的只算一条消息

func GetTotalCountForActorMsgs(ctx context.Context, actor string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgs")

	api := fullnode.API.GetAppropriateAPI()

	addr, err := address.NewFromString(buildnet.NetPrefix + actor)
	if err != nil {
		return nil, err
	}

	actorID := actor
	switch addr.Protocol() {
	case address.ID:
		actorID = addr.String()[1:]
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK) //todo: 缓存
		if err != nil {
			return nil, err
		}

		actorID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for actor %v", actor)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// transfer_count_for_actor2.js
func GetTotalCountForActorTransferMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgs")

	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorEvents(ctx context.Context, actor string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	api := fullnode.API.GetAppropriateAPI()

	addr, err := address.NewFromString(buildnet.NetPrefix + actor)
	if err != nil {
		return nil, err
	}

	actorID := actor
	switch addr.Protocol() {
	case address.ID:
		actorID = addr.String()[1:]
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		actorID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for actor %v", actor)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorEvents)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// 出块列表 BlockHeader查，现在就是这样
func GetTotalCountForMinedMsgsMap(ctx context.Context, minerID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForMinedMsgsMap")

	countUtils, err := refresh(ctx, dbsm, curEpoch, minerID, "", refreshTotalCountForMinedMsgsMap)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// count_of_largeamount_transfers.js
func GetTotalCountForTransfersForLargeAmount(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgs")

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshTotalCountForTransfersForLargeAmount)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorTransferBlockRewardMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferBlockRewardMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorTransferBurnMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferBurnMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorTransferSendAndReceiveMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferSendAndReceiveMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorTransferSendMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferSendMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForActorTransferReceiveMsgs(ctx context.Context, actorID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, actorID, "", refreshTotalCountForActorTransferReceiveMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetAllBlockMsgsByMethodName(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) (map[string]int64, error) {
	curTime := time.Now()
	alk.RLock()
	if curTime.Sub(lastUpdateTime) < 24*time.Hour {
		defer alk.RUnlock()
		return AllMethods, nil
	}

	alk.RUnlock()

	blockMsgsByMethodNames := make([]string, 0)

	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	dbs := make([]common.DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		dsn := db.Url()
		state, found, err := dbsm.GetState(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if !found {
			return nil, fmt.Errorf("state of dsn %v not found", dsn)
		}

		cols, ok := dbsm.GetDBCollections(dsn)
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
		}

		switch state.GetDType() {
		case smodel.Formal:
			allBlockMethodNames, err := GetAllBlockMethodNames(ctx, state, cols)
			if err != nil {
				return nil, fmt.Errorf("GetAllBlockMethodNames for db %v failed: %v", tmp.Url(), err)
			}

			if len(allBlockMethodNames) != 0 {
				blockMsgsByMethodNames = append(blockMsgsByMethodNames, allBlockMethodNames[0].MethodNames...)
			}

			tmpStartEpoch = state.GetEndEpoch()
		case smodel.Cold:
			allBlockMethodNames, err := GetAllBlockMethodNames(ctx, state, cols)
			if err != nil {
				return nil, fmt.Errorf("GetAllBlockMethodNames for db %v failed: %v", tmp.Url(), err)
			}

			if len(allBlockMethodNames) != 0 {
				blockMsgsByMethodNames = append(blockMsgsByMethodNames, allBlockMethodNames[0].MethodNames...)
			}
		default:
			return nil, fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		dsn := tmp.Url()
		state := segment.DefaultState(dsn, smodel.Tmp, smodel.DefaultInterval, tmpStartEpoch, curEpoch+1, 0, 0)

		cols, ok := dbsm.GetDBCollections(dsn)
		if !ok {
			return nil, fmt.Errorf("dsn %v not found in DBCollectionsMap", dsn)
		}

		allBlockMethodNames, err := GetAllBlockMethodNames(ctx, state, cols)
		if err != nil {
			return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
		}

		if len(allBlockMethodNames) != 0 {
			blockMsgsByMethodNames = append(blockMsgsByMethodNames, allBlockMethodNames[0].MethodNames...)
		}
	}

	MethodNamesMap := make(map[string]int64, 0)
	for _, methodName := range blockMsgsByMethodNames {
		if _, ok := MethodNamesMap[methodName]; !ok {
			MethodNamesMap[methodName] = 1
		}
	}

	alk.Lock()
	AllMethods = MethodNamesMap
	lastUpdateTime = time.Now()
	alk.Unlock()

	return MethodNamesMap, nil
}

func GetAllBlockMethodNamesMap(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols common.Collections) ([]model.AllMethodsForActorRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allBlockMethodNamesRes []model.AllMethodsForActorRes

	js := "// ExecTrace\n[\n    {\n        $match: {\n            \"IsBlock\": true,\n            \"Epoch\": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},\n        }\n    },\n    {\n        $group: {\n            _id: \"$Msg.MethodName\",\n            Count:{$sum:1}\n        }\n    }\n]"
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, js)
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allBlockMethodNamesRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allBlockMethodNamesRes, nil
		}
	}

	return nil, fmt.Errorf("no ExecTrace collection")
}

func GetAllActorMsgsByMethodNameMap(ctx context.Context, dbsm *DataBaseStateManager, actorID string, curEpoch abi.ChainEpoch) (map[string]int64, error) {
	actorMsgsByMethodNameMap := make(map[string]int64)

	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	dbs := make([]common.DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		dsn := db.Url()
		state, found, err := dbsm.GetState(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if !found {
			return nil, fmt.Errorf("state of dsn %v not found", dsn)
		}

		cols, ok := dbsm.GetDBCollections(dsn)
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
		}

		switch state.GetDType() {
		case smodel.Formal:
			actorMethodsMap, err := GetAllActorMethodStates(ctx, state, cols, actorID)
			if err != nil {
				return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
			}

			for _, actorMethod := range actorMethodsMap {
				actorMsgsByMethodNameMap[actorMethod.MethodName] += actorMethod.Count
			}

			tmpStartEpoch = state.GetEndEpoch()
		case smodel.Cold:
			actorMethodsMap, err := GetAllActorMethodStates(ctx, state, cols, actorID)
			if err != nil {
				return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
			}

			for _, actorMethod := range actorMethodsMap {
				actorMsgsByMethodNameMap[actorMethod.MethodName] += actorMethod.Count
			}
		default:
			return nil, fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		dsn := tmp.Url()
		state := segment.DefaultState(dsn, smodel.Tmp, smodel.DefaultInterval, tmpStartEpoch, curEpoch+1, 0, 0)

		cols, ok := dbsm.GetDBCollections(dsn)
		if !ok {
			return nil, fmt.Errorf("dsn %v not found in DBCollectionsMap", dsn)
		}

		actorMethodsMap, err := GetAllActorMethodStates(ctx, state, cols, actorID)
		if err != nil {
			return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
		}

		for _, actorMethod := range actorMethodsMap {
			actorMsgsByMethodNameMap[actorMethod.MethodName] += actorMethod.Count
		}
	}

	return actorMsgsByMethodNameMap, nil
}

func GetAllActorMethods(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, actorID string, cols common.Collections) ([]model.AllMethodsForActorRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allMethodsForActorRes []model.AllMethodsForActorRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetAllMethodNamesForActor()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allMethodsForActorRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allMethodsForActorRes, nil
		}
	}

	return nil, fmt.Errorf("no ActorMessage collection")
}

func GetAllActorsMethods(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols common.Collections) ([]model.AllActorsMethodsRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allActorsMethodsRes []model.AllActorsMethodsRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetAllMethodNamesForActorsAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allActorsMethodsRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allActorsMethodsRes, nil
		}
	}

	return nil, fmt.Errorf("no ActorMessage collection")
}

func GetAllActorsMsgsCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols common.Collections) ([]model.AllActorsMsgsCountRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allActorsMsgsCountRes []model.AllActorsMsgsCountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetAllActorsMsgsCountAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allActorsMsgsCountRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allActorsMsgsCountRes, nil
		}
	}

	return nil, fmt.Errorf("no ActorMessage collection")
}

func GetAllActorsTransferMsgsCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols common.Collections) ([]model.AllActorsMsgsCountRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allActorsTransferMsgsCountRes []model.AllActorsMsgsCountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetTransferMessagesAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allActorsTransferMsgsCountRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allActorsTransferMsgsCountRes, nil
		}
	}

	return nil, fmt.Errorf("no ActorMessage collection")
}

func GetAllMinersMinedCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols common.Collections) ([]model.AllActorsMsgsCountRes, error) {
	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allMinersMinedCount []model.AllActorsMsgsCountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetAllMinersMinedCountAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "BlockHeader"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allMinersMinedCount)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allMinersMinedCount, nil
		}
	}

	return nil, fmt.Errorf("no BlockHeader collection")
}

func refreshEpochRange(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	switch state.GetDType() {
	case smodel.Formal:
		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, DType: state.GetDType()})
		// todo: *tmpStartEpoch = state.GetEndEpoch() 保证所有状态做完才更新EndEpoch
		*tmpStartEpoch = state.GetEndEpoch()

		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForTipSets(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetTipSetStates(ctx, state, cols, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, TipSetStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetTipSetStates(ctx, state, cols, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, TipSetStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetTipSetStates(ctx, state, cols, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, TipSetStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshDealRange(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	switch state.GetDType() {
	case smodel.Formal, smodel.Cold:
		startDealID, endDealID, err := GetDealIDRange(ctx, cols, int64(state.GetStartEpoch()), int64(state.GetEndEpoch()))
		if err != nil {
			return err
		}

		count := endDealID - startDealID

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(startDealID), End: int64(endDealID), DealState: int64(count), Cols: cols, DType: state.GetDType()})
		return nil
	case smodel.Tmp:
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForBlockMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		blockStates, err := DBStateManager.GetBlockStates(ctx, state.GetDSN())
		if err != nil {
			return err
		}

		// todo: 后面blockStates排序会影响写入乱吗？
		sortBlockStates := make([]smodel.SegmentState, len(blockStates))
		n := copy(sortBlockStates, blockStates)
		if n != len(blockStates) {
			return fmt.Errorf("copy blockStates failed: copied(%v)/total(%v)", n, len(blockStates))
		}

		sort.Slice(sortBlockStates, func(i, j int) bool {
			return sortBlockStates[i].StartEpoch > sortBlockStates[j].StartEpoch
		})

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), BlockStates: sortBlockStates, Cols: cols, DType: state.GetDType()})
		if len(sortBlockStates) > 0 {
			*tmpStartEpoch = sortBlockStates[0].EndEpoch
		} else {
			*tmpStartEpoch = state.GetStartEpoch()
		}
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		blockStates, err := GetBlockStates(ctx, state, cols, actorID, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), BlockStates: blockStates, Cols: cols, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		blockStates, err := DBStateManager.GetBlockStates(ctx, state.GetDSN())
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), BlockStates: blockStates, Cols: cols, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

// actor筛选的，旧库不缓存？
func refreshTotalCountForActorDeals(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal, smodel.Cold:
		// todo: formal state 每次拿
		startDealID, endDealID, err := GetDealIDRange(ctx, cols, int64(state.GetStartEpoch()), int64(state.GetEndEpoch()))
		if err != nil {
			return err
		}

		state.SetDealState(smodel.DealState{StartDealID: startDealID, EndDealID: endDealID})

		count, err := GetDealActorStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(startDealID), End: int64(endDealID), DealActorStates: count, Cols: cols, DType: state.GetDType()})

		return nil
	case smodel.Tmp:
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

// todo: colds存下永久状态 或 缓存
func refreshTotalCountForBlockMsgsByMethodName(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		blockMethodStates, err := DBStateManager.GetBlockMethodStates(ctx, state.GetDSN(), methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, BlockMethodStates: blockMethodStates, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		blockMethodStates, err := GetBlockMethodStates(ctx, state, cols, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, BlockMethodStates: blockMethodStates, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		blockMethodStates, err := DBStateManager.GetBlockMethodStates(ctx, state.GetDSN(), methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, BlockMethodStates: blockMethodStates, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorMsgByMethodName(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorMethodStates(ctx, state, cols, actorID, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorMethodStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorMethodStates(ctx, state, cols, actorID, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorMethodStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorMethodStates(ctx, state, cols, actorID, methodName)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorMethodStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorEvents(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorEventStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorEventStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorEventStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorEventStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorEventStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorEventStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForMinedMsgsMap(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetMinedStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, MinedStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetMinedStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, MinedStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetMinedStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, MinedStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForTransfersForLargeAmount(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetLargeAmountTransferStates(ctx, state, cols)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, LargeAmountTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetLargeAmountTransferStates(ctx, state, cols)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, LargeAmountTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetLargeAmountTransferStates(ctx, state, cols)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, LargeAmountTransferStates: count, DType: state.GetDType()})
		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferBlockRewardMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferBlockRewardStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferBlockRewardStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferBlockRewardStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})

		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferBurnMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferBurnStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferBurnStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferBurnStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})

		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferSendAndReceiveMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferSendAndReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferSendAndReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferSendAndReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})

		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferSendMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferSendStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferSendStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferSendStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})

		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func refreshTotalCountForActorTransferReceiveMsgs(ctx context.Context, state *segment.State, cols common.Collections, concurrentCountUtils *ConcurrentCountUtils, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	switch state.GetDType() {
	case smodel.Formal:
		count, err := GetActorTransferReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		*tmpStartEpoch = state.GetEndEpoch()
		return nil
	case smodel.Tmp:
		state.SetStartEpoch(*tmpStartEpoch)
		state.SetEndEpoch(curEpoch + 1)

		count, err := GetActorTransferReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})
		return nil
	case smodel.Cold:
		count, err := GetActorTransferReceiveStates(ctx, state, cols, actorID)
		if err != nil {
			return err
		}

		concurrentCountUtils.lk.Lock()
		defer concurrentCountUtils.lk.Unlock()
		concurrentCountUtils.CountUtils = append(concurrentCountUtils.CountUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch()), Cols: cols, ActorTransferStates: count, DType: state.GetDType()})

		return nil
	default:
		return fmt.Errorf("invalid dtype: %v for dsn: %v", state.GetDType(), state.GetDSN())
	}
}

func GetTipSetStates(ctx context.Context, state *segment.State, cols common.Collections, methodName string) (int64, error) {
	rlog := log.With("query", "GetTipSetStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("get BlockMethodStates successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), MethodName: methodName}, string(monitor.GetCountOfTipsetAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "Tipset"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	// log.Warnf

	return 0, fmt.Errorf("no Tipset collection")
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
// only for tmp, get from londobell
func GetBlockStates(ctx context.Context, state *segment.State, cols common.Collections, actorID, methodName string) ([]smodel.SegmentState, error) {
	rlog := log.With("query", "GetBlockStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh BlockMsgsCount successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return nil, nil
	}

	blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: startEpoch}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: endEpoch}}}, {Key: "IsBlock", Value: true}}

	var (
		count int64
		err   error
	)
	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err = col.CountDocuments(ctx, blockFilter)
			if err != nil {
				return nil, err
			}

			log.Infow("RefreshBlockMsgs", "count", count)
			return []smodel.SegmentState{smodel.SegmentState{
				Dsn:        state.GetDSN(),
				StartEpoch: startEpoch,
				EndEpoch:   endEpoch,
				Count:      count,
			}}, nil
		}
	}

	return nil, fmt.Errorf("no ExecTrace collection")
}

func GetBlockMethodStates(ctx context.Context, state *segment.State, cols common.Collections, methodName string) ([]smodel.SegmentState, error) {
	rlog := log.With("query", "GetBlockMethodStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("get BlockMethodStates successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return nil, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), MethodName: methodName}, string(monitor.GetCountOfBlockMessagesByMethodNameAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			if len(countRes) == 0 {
				return nil, nil
			}

			return []smodel.SegmentState{smodel.SegmentState{
				Dsn:        state.GetDSN(),
				StartEpoch: startEpoch,
				EndEpoch:   endEpoch,
				Count:      countRes[0].Count,
			}}, nil
		}
	}

	// log.Warnf

	return nil, fmt.Errorf("no ExecTrace collection")
}

// todo: 第二轮优化
func GetAllBlockMethodNames(ctx context.Context, state *segment.State, cols common.Collections) ([]model.AllBlockMethodNamesRes, error) {
	rlog := log.With("query", "GetAllBlockMethodNames")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("GetAllBlockMethodNames successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allBlockMethodNamesRes []model.AllBlockMethodNamesRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetAllBlockMethodNamesAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allBlockMethodNamesRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allBlockMethodNamesRes, nil
		}
	}

	return nil, fmt.Errorf("no ExecTrace collection")
}

func GetActorStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh ActorMsgsCountMap successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfMessageForActorAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := ctx.Value(TableKey).(string)
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) // todo
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorMethodStates(ctx context.Context, state *segment.State, cols common.Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("query", "GetActorMethodStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh ActorMsgsByMethodNameMap successfully [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID, MethodName: methodName}, string(monitor.GetCountOfActorMessagesByMethodNameAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := ctx.Value(TableKey).(string)
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil

		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetAllActorMethodStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) ([]model.AllMethodsForActorRes, error) {
	rlog := log.With("query", "GetAllActorMethodStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("GetAllActorMethodStates successfully [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return nil, nil
	}

	var allMethodsForActorRes []model.AllMethodsForActorRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetAllMethodNamesForActor()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return nil, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			err = cur.All(ctx, &allMethodsForActorRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return nil, err
			}

			return allMethodsForActorRes, nil
		}
	}

	return nil, fmt.Errorf("no ActorMessage collection")
}

func GetActorTransferStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh ActorTransfersCountMap successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransfersForActor2Aggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) // todo
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorEventStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorEventStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh ActorEventStates successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfEventsForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorEvent"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorEvent collection")
}

func GetMinedStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetMinedStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh MinedMsgsMap successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetMinedCountForMinersAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "BlockHeader"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no BlockHeader collection")
}

func GetLargeAmountTransferStates(ctx context.Context, state *segment.State, cols common.Collections) (int64, error) {
	rlog := log.With("query", "GetLargeAmountTransferStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {

		rlog.Infof("refresh TransfersLargeAmountCount successfully for [%v, %v), elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch)}, string(monitor.GetCountOfLargeAmountTransfersAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no large amount transfers between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ExecTrace collection")
}

func GetDealActorStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "DealActorStates")

	start := time.Now()
	startDealID, endDealID := state.GetDealStartID(), state.GetDealEndID()
	defer func() {
		rlog.Infof("refresh DealActorStates successfully for [%v, %v), elapsed: %v", startDealID, endDealID, time.Now().Sub(start).String())
	}()

	var countRes []model.CountRes

	pipe, err := util.Parse(model.Ctx{Start: int64(startDealID), End: int64(endDealID), Addr: actorID}, string(monitor.GetCountOfDealsByAddrAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "NewDealProposal"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no deal for actor %v for [%v, %v)", actorID, startDealID, endDealID)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no DealProposal collection")
}

func GetActorTransferBlockRewardStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferBlockRewardStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh actorTransferBlockRewardStates successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransferBlockRewardForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorTransferBurnStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferBurnStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh actorTransferBurnStates successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransferBurnForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorTransferSendAndReceiveStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferSendAndReceiveStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh actorTransferSendAndReceiveStates successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransferSendAndReceiveForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorTransferSendStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferSendStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh actorTransferSendStates successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransferSendForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetActorTransferReceiveStates(ctx context.Context, state *segment.State, cols common.Collections, actorID string) (int64, error) {
	rlog := log.With("query", "GetActorTransferReceiveStates")

	start := time.Now()
	startEpoch, endEpoch := state.GetStartEpoch(), state.GetEndEpoch()
	defer func() {
		rlog.Infof("refresh actorTransferReceiveStates successfully for [%v, %v], elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if endEpoch <= startEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(startEpoch), EndEpoch: int64(endEpoch), Addr: actorID}, string(monitor.GetCountOfTransferReceiveForActorAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			err = cur.All(ctx, &countRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return 0, err
			}

			if len(countRes) == 0 {
				rlog.Warnf("no actor between %v and %v", startEpoch, endEpoch)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func GetIDForAddr(ctx context.Context, from, to string, log *zap.SugaredLogger, api v0api.FullNode) (fromID, toID string, fromErr, toErr bool) {
	// todo: 消息中有无效的地址暂时跳过不存储
	fromAddr, err := address.NewFromString(buildnet.NetPrefix + from)
	if err != nil {
		log.Warnf("invalid from address: %v", from)
		fromErr = true
	}

	if !fromErr {
		if fromAddr.Protocol() == address.ID {
			fromID = fromAddr.String()[1:]
		} else if fromAddr.Protocol() == address.SECP256K1 || fromAddr.Protocol() == address.Actor || fromAddr.Protocol() == address.BLS || fromAddr.Protocol() == address.Delegated {
			ALock.Lock()
			if _, ok := ActorIDMap[fromAddr.String()[1:]]; !ok {
				ID, err := api.StateLookupID(ctx, fromAddr, types.EmptyTSK)
				if err != nil {
					log.Warnf("lookup ID for from address %v failed: %v", from, err)
					fromErr = true
				}

				fromID = ID.String()[1:]

				ActorIDMap[fromAddr.String()[1:]] = fromID
			} else {
				// map缓存
				fromID = ActorIDMap[fromAddr.String()[1:]]
			}
			ALock.Unlock()
		}
	}

	toAddr, err := address.NewFromString(buildnet.NetPrefix + to)
	if err != nil {
		log.Warnf("invalid from address: %v", from)
		toErr = true
	}

	if !toErr {
		if toAddr.Protocol() == address.ID {
			toID = toAddr.String()[1:]
		} else if toAddr.Protocol() == address.SECP256K1 || toAddr.Protocol() == address.Actor || toAddr.Protocol() == address.BLS || toAddr.Protocol() == address.Delegated {
			ALock.Lock()
			if _, ok := ActorIDMap[toAddr.String()[1:]]; !ok {
				ID, err := api.StateLookupID(ctx, toAddr, types.EmptyTSK)
				if err != nil {
					log.Warnf("lookup ID for to address %v failed: %v", to, err)
					toErr = true
				}

				toID = ID.String()[1:]

				ActorIDMap[toAddr.String()[1:]] = toID
			} else {
				// map缓存
				toID = ActorIDMap[toAddr.String()[1:]]
			}
			ALock.Unlock()
		}
	}

	return fromID, toID, fromErr, toErr
}

func GetColsOnly(dbsm *DataBaseStateManager) ([]CountUtil, error) {
	countUtils := make([]CountUtil, 0)

	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	if !tmp.IsInvalidDB() {
		cols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		countUtils = append(countUtils, CountUtil{Cols: cols, DType: smodel.Tmp})
	}

	if !formal.IsInvalidDB() {
		cols, ok := dbsm.GetDBCollections(formal.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
		}

		countUtils = append(countUtils, CountUtil{Cols: cols, DType: smodel.Formal})
	}

	dbs := make([]common.DB, len(colds))
	dbs = append(dbs, colds...)

	// cold db状态已加载
	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		cols, ok := dbsm.GetDBCollections(db.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
		}

		countUtils = append(countUtils, CountUtil{Cols: cols, DType: smodel.Cold})
	}

	// 不需要排序

	return countUtils, nil
}
