package multiquery

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
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
	//refreshOnce sync.Once
	//refreshes   = make([]func(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error), 0)

	addupOnce sync.Once
	addupes   = make([]func(ctx context.Context, ds *DataBaseState, cols Collections) error, 0)

	AllMethods     = make(map[string]int64, 0)
	alk            sync.RWMutex
	lastUpdateTime time.Time
)

func init() {
	//refreshOnce.Do(func() {
	//	refreshes = append(refreshes, RefreshBlockMsgs, RefreshBlockMsgsByMethodName, RefreshActorMsgsByMethodName, RefreshActorMsgs, RefreshActorTransferMsgs, RefreshMinedMsgsMaps, RefreshTransfersForLargeAmount)
	//})

	addupOnce.Do(func() {
		addupes = append(addupes, AddUpBlockMsgsCount)
	})

}

func PeriodicRefreshDataBaseState(ctx context.Context, dbsm *DataBaseStateManager) {
	tick := time.NewTicker(15 * time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			finalHeight, err := GetFinalHeightForFormalDB(ctx, dbsm)
			if err != nil {
				log.Error(err)
				continue
			}

			start := time.Now()
			log.Infow("begin PeriodicRefreshDataBaseState for formal, finalHeight: %v", finalHeight)
			if err := RefreshFormalDataBaseState(ctx, dbsm, finalHeight); err != nil {
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
func RefreshFormalDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, finalHeight abi.ChainEpoch) error {
	formal := dbsm.GetFormalCfg()

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
		return nil
	}

	dbState, found, err := dbsm.Stm.LoadDataBaseState(formal.Url())
	if err != nil {
		return err
	}

	if !found {
		////todo
		//dbState, err := InitDataBaseState(ctx, db, true)
		//if err != nil {
		//	return 0, nil, err
		//}
		//
		//fmt.Println("InitDataBaseState", dbState)
		//if err := m.SetDataBaseState(db.Url, *dbState); err != nil {
		//	return 0, nil, err
		//}
		return fmt.Errorf("db %v not found", formal)
	}

	cols, ok := dbsm.GetDBCollections(formal.Url())
	if !ok {
		return fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
	}

	dbState.EndEpoch = finalHeight + 1

	var ewg multierror.Group
	for i := range addupes {
		i := i
		addup := addupes[i]
		ewg.Go(func() error {
			if err := addup(ctx, &dbState, cols); err != nil {
				return err
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		log.Errorf("RefreshFormalDataBaseState failed: %v", err)
		return err
	}

	//if err := RefreshBlockMsgs(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	//if err := RefreshBlockMsgsByMethodName(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	//if err := RefreshActorMsgsByMethodName(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	//if err := RefreshActorMsgs(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	//if err := RefreshActorTransferMsgs(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	//if err := RefreshMinedMsgsMaps(ctx, &dbState, cols); err != nil {
	//	return err
	//}
	////if err := RefreshTransfersForLargeAmount(ctx, &dbState, cols); err != nil { // todo
	////	return err
	////}
	//

	if err := dbsm.Stm.SetDataBaseState(formal.Url(), dbState); err != nil {
		return err
	}

	dbsm.DBStateCache.SetDataBase(formal.Url(), &dbState)

	log.Infof("RefreshFormalDataBaseState successfully, dbState.EndEpoch: %v", finalHeight+1)

	return nil
}

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

func refresh(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch, actorID, methodName string, f func(context.Context, *DataBaseState, Collections, *[]CountUtil, *abi.ChainEpoch, abi.ChainEpoch, string, string) error) ([]CountUtil, error) {
	countUtils := make([]CountUtil, 0)

	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	dbs := make([]DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		dbState, err := dbsm.GetDataBase(db.Url())
		if err != nil {
			return nil, err
		}

		cols, ok := dbsm.GetDBCollections(db.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
		}

		err = f(ctx, dbState, cols, &countUtils, &tmpStartEpoch, curEpoch, actorID, methodName)
		if err != nil {
			return nil, err
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpDBState := DefaultDataBaseState(false, true, 0, 0)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		err := f(ctx, tmpDBState, tmpCols, &countUtils, &tmpStartEpoch, curEpoch, actorID, methodName)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(countUtils, func(i, j int) bool {
		return countUtils[i].End > countUtils[j].End
	})

	return countUtils, nil
}

func GetEpochRange(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshEpochRange)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

func GetTotalCountForBlockMsgs(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForBlockMsgs")

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

//count_of_actormessages_by_methodname.js
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

//出块列表 BlockHeader查，现在就是这样
func GetTotalCountForMinedMsgsMap(ctx context.Context, minerID string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForMinedMsgsMap")

	countUtils, err := refresh(ctx, dbsm, curEpoch, minerID, "", refreshTotalCountForMinedMsgsMap)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

//count_of_largeamount_transfers.js
func GetTotalCountForTransfersForLargeAmount(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgs")

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", "", refreshTotalCountForTransfersForLargeAmount)
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

	dbs := make([]DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		dbState, err := dbsm.GetDataBase(db.Url())
		if err != nil {
			return nil, err
		}

		if dbState.Formal {
			cols, ok := dbsm.GetDBCollections(db.Url())
			if !ok {
				return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
			}

			finalHeight, err := GetFinalHeight(ctx, cols)
			if err != nil {
				return nil, err
			}

			dbState.EndEpoch = finalHeight + 1

			allBlockMethodNames, err := GetAllBlockMethodNames(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
			if err != nil {
				return nil, fmt.Errorf("GetAllBlockMethodNames for db %v failed: %v", tmp.Url(), err)
			}

			if len(allBlockMethodNames) != 0 {
				blockMsgsByMethodNames = append(blockMsgsByMethodNames, allBlockMethodNames[0].MethodNames...)
			}

			tmpStartEpoch = dbState.EndEpoch
		} else {
			for methodName := range dbState.BlockMsgsByMethodNameMap {
				blockMsgsByMethodNames = append(blockMsgsByMethodNames, methodName)
			}
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpDBState := DefaultDataBaseState(false, true, tmpStartEpoch, curEpoch+1)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		allBlockMethodNames, err := GetAllBlockMethodNames(ctx, tmpDBState.StartEpoch, tmpDBState.EndEpoch, tmpCols)
		if err != nil {
			return nil, fmt.Errorf("GetAllBlockMethodNames for db %v failed: %v", tmp.Url(), err)
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

func GetAllBlockMethodNamesMap(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllMethodsForActorRes, error) {
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

func GetAllBlockMethodNames(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllBlockMethodNamesRes, error) {
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

func GetAllActorMsgsByMethodNameMap(ctx context.Context, dbsm *DataBaseStateManager, actorID string, curEpoch abi.ChainEpoch) (map[string]int64, error) {
	actorMsgsByMethodNameMap := make(map[string]int64)

	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	dbs := make([]DB, 0)
	dbs = append(dbs, colds...)
	dbs = append(dbs, formal)

	var tmpStartEpoch abi.ChainEpoch

	for _, db := range dbs {
		if db.IsInvalidDB() {
			continue
		}

		dbState, err := dbsm.GetDataBase(db.Url())
		if err != nil {
			return nil, err
		}

		if dbState.Formal {
			cols, ok := dbsm.GetDBCollections(db.Url())
			if !ok {
				return nil, fmt.Errorf("url %v not found in DBCollectionsMap", db.Url())
			}

			finalHeight, err := GetFinalHeight(ctx, cols)
			if err != nil {
				return nil, err
			}

			dbState.EndEpoch = finalHeight + 1

			actorMethodsMap, err := GetAllActorMethods(ctx, dbState.StartEpoch, dbState.EndEpoch, actorID, cols)
			if err != nil {
				return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
			}

			for _, actorMethod := range actorMethodsMap {
				actorMsgsByMethodNameMap[actorMethod.MethodName] += actorMethod.Count
			}

			tmpStartEpoch = dbState.EndEpoch
		} else {
			for methodName, actorMsgsCountMap := range dbState.ActorMsgsByMethodNameMap {
				if count, ok := actorMsgsCountMap[actorID]; ok {
					actorMsgsByMethodNameMap[methodName] += count
				}
			}
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpDBState := DefaultDataBaseState(false, true, tmpStartEpoch, curEpoch+1)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		actorMethodsMap, err := GetAllActorMethods(ctx, tmpDBState.StartEpoch, tmpDBState.EndEpoch, actorID, tmpCols)
		if err != nil {
			return nil, fmt.Errorf("GetAllActorMethods for db %v failed: %v", tmp.Url(), err)
		}

		for _, actorMethod := range actorMethodsMap {
			actorMsgsByMethodNameMap[actorMethod.MethodName] += actorMethod.Count
		}
	}

	return actorMsgsByMethodNameMap, nil
}

func GetAllActorMethods(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, actorID string, cols Collections) ([]model.AllMethodsForActorRes, error) {
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

func GetAllActorsMethods(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllActorsMethodsRes, error) {
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

func GetAllActorsMsgsCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllActorsMsgsCountRes, error) {
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

func GetAllActorsTransferMsgsCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllActorsMsgsCountRes, error) {
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

func GetAllMinersMinedCount(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, cols Collections) ([]model.AllActorsMsgsCountRes, error) {
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

func refreshEpochRange(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols})
		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols})
		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols})
	return nil
}

func refreshTotalCountForBlockMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		if dbState.NextEpochForBlockMsgsCount > dbState.StartEpoch {
			*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsCount, Cols: cols})
			*tmpStartEpoch = dbState.NextEpochForBlockMsgsCount
			return nil
		}

		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshBlockMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})
		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshBlockMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})
		return nil
	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsCount, Cols: cols})
	return nil
}

func refreshTotalCountForBlockMsgsByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshBlockMsgsByMethodName(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})
		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshBlockMsgsByMethodName(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})
		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsByMethodNameMap[methodName], Cols: cols})
	return nil
}

func refreshTotalCountForActorMsgByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshActorMsgsByMethodName(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshActorMsgsByMethodName(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		return nil

	}

	count := int64(0)
	if _, ok := dbState.ActorMsgsByMethodNameMap[methodName]; ok {
		count = dbState.ActorMsgsByMethodNameMap[methodName][actorID]
	}
	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

	return nil
}

func refreshTotalCountForActorMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1
		count, err := RefreshActorMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshActorMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorMsgsCountMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForActorTransferMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshActorTransferMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshActorTransferMsgs(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorTransfersCountMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForMinedMsgsMap(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshMinedMsgsMaps(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshMinedMsgsMaps(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.MinedMsgsMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForTransfersForLargeAmount(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, actorID, methodName string) error {
	if dbState.Formal {
		finalHeight, err := GetFinalHeight(ctx, cols)
		if err != nil {
			return err
		}

		dbState.EndEpoch = finalHeight + 1

		count, err := RefreshTransfersForLargeAmount(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		count, err := RefreshTransfersForLargeAmount(ctx, dbState, cols, actorID, methodName)
		if err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.TransfersLargeAmountCount, Cols: cols})

	return nil
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
func RefreshBlockMsgs(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "BlockMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh BlockMsgsCount successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: ds.StartEpoch}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: ds.EndEpoch}}}, {Key: "IsBlock", Value: true}}

	var (
		count int64
		err   error
	)
	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err = col.CountDocuments(ctx, blockFilter)
			if err != nil {
				return 0, err
			}

			log.Infow("RefreshBlockMsgs", "count", count)
			return count, nil
		}
	}

	return 0, fmt.Errorf("no ExecTrace collection")
}

func RefreshBlockMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "BlockMsgsByMethodName")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh BlockMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch), MethodName: methodName}, string(monitor.GetCountOfBlockMessagesByMethodNameAggregator()))
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
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	// log.Warnf

	return 0, fmt.Errorf("no ExecTrace collection")
}

func RefreshActorMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "ActorMsgsByMethodName")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch), Addr: actorID, MethodName: methodName}, string(monitor.GetCountOfActorMessagesByMethodNameAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return 0, err
	}

	tableName := "ActorMessage"
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
				rlog.Warnf("no actor between %v and %v", ds.StartEpoch, ds.EndEpoch-1)
				return 0, nil
			}

			return countRes[0].Count, nil

		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func RefreshActorMsgs(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "ActorMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorMsgsCountMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch), Addr: actorID}, string(monitor.GetCountOfMessageForActorAggregator())) // todo: ExecTrace
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
				rlog.Warnf("no actor between %v and %v", ds.StartEpoch, ds.EndEpoch-1)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func RefreshActorTransferMsgs(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "ActorTransferMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorTransfersCountMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch), Addr: actorID}, string(monitor.GetCountOfTransfersForActor2Aggregator())) // todo: ExecTrace
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
				rlog.Warnf("no actor between %v and %v", ds.StartEpoch, ds.EndEpoch-1)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ActorMessage collection")
}

func RefreshMinedMsgsMaps(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "MinedMsgsMap")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh MinedMsgsMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch), Addr: actorID}, string(monitor.GetMinedCountForMinersAggregator())) // todo: ExecTrace
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
				rlog.Warnf("no actor between %v and %v", ds.StartEpoch, ds.EndEpoch-1)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no BlockHeader collection")
}

func RefreshTransfersForLargeAmount(ctx context.Context, ds *DataBaseState, cols Collections, actorID, methodName string) (int64, error) {
	rlog := log.With("refresh", "TransfersForLargeAmount")

	start := time.Now()
	startEpoch, endEpoch := ds.StartEpoch, ds.EndEpoch-1
	defer func() {

		rlog.Infof("refresh TransfersLargeAmountCount successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.StartEpoch {
		return 0, nil
	}

	var countRes []model.CountRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.StartEpoch), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetCountOfLargeAmountTransfersAggregator())) // todo: ExecTrace
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
				rlog.Warnf("no large amount transfers between %v and %v", ds.StartEpoch, ds.EndEpoch-1)
				return 0, nil
			}

			return countRes[0].Count, nil
		}
	}

	return 0, fmt.Errorf("no ExecTrace collection")
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

		countUtils = append(countUtils, CountUtil{Cols: cols, Tmp: true})
	}

	if !formal.IsInvalidDB() {
		cols, ok := dbsm.GetDBCollections(formal.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
		}

		countUtils = append(countUtils, CountUtil{Cols: cols, Formal: true})
	}

	dbs := make([]DB, len(colds))
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

		countUtils = append(countUtils, CountUtil{Cols: cols})
	}

	// 不需要排序

	return countUtils, nil
}

// 累加
func AddUpBlockMsgsCount(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("AddUp", "BlockMsgsCount")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForBlockMsgsCount, ds.EndEpoch-1

	var (
		count int64
		err   error
	)

	defer func() {
		rlog.Infof("addup BlockMsgsCount successfully between %v and %v, count: %v, elapsed: %v", startEpoch, endEpoch, ds.BlockMsgsCount, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForBlockMsgsCount {
		return nil
	}

	blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: ds.NextEpochForBlockMsgsCount}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: ds.EndEpoch}}}, {Key: "IsBlock", Value: true}}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err = col.CountDocuments(ctx, blockFilter)
			if err != nil {
				return err
			}

			ds.BlockMsgsCount += count
			ds.NextEpochForBlockMsgsCount = ds.EndEpoch

			return nil
		}
	}

	return fmt.Errorf("no ExecTrace collection")
}
