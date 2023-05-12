package multiquery

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/filecoin-project/lotus/api/v0api"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"go.mongodb.org/mongo-driver/bson"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/buildnet"
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
	refreshOnce sync.Once
	refreshes   = make([]func(ctx context.Context, ds *DataBaseState, cols Collections) error, 0)
)

func init() {
	refreshOnce.Do(func() {
		refreshes = append(refreshes, RefreshBlockMsgs, RefreshBlockMsgsByMethodName, RefreshActorMsgsByMethodName, RefreshActorMsgs, RefreshActorTransferMsgs, RefreshMinedMsgsMaps /*RefreshTransfersForLargeAmount*/)
	})
}

func PeriodicRefreshDataBaseState(ctx context.Context, dbsm *DataBaseStateManager) {
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			res, err := GetFinalHeightForFormalDB(ctx, dbsm)
			if err != nil {
				log.Error(err)
				continue
			}

			if len(res) == 0 { // todo
				log.Warnf("no data in FinalHeight")
				continue
			}

			start := time.Now()
			log.Infow("begin PeriodicRefreshDataBaseState for formal, finalHeight: %v", res[0].Epoch)
			if err := RefreshFormalDataBaseState(ctx, dbsm, res[0].Epoch); err != nil {
				log.Error(err)
				continue
			}

			log.Infof("finish PeriodicRefreshDataBaseState for formal, elapsed: %v", time.Now().Sub(start).String())
		}
	}
}

func TestPeriodicRefreshDataBaseState(ctx context.Context, dbsm *DataBaseStateManager) {
	tick := time.NewTicker(10 * time.Minute) //todo:test
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			//res, err := GetFinalHeightForFormalDB(ctx, dbsm)
			//if err != nil {
			//	log.Error(err)
			//	continue
			//}
			//
			//if len(res) == 0 { // todo
			//	log.Warnf("no data in FinalHeight")
			//	continue
			//}

			//todo:test
			if err := TestRefreshFormalDataBaseState(ctx, dbsm, IncrementalEndEpoch); err != nil {
				log.Error(err)
				continue
			}
			IncrementalEndEpoch += 60
		}
	}
}

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
	for i := range refreshes {
		i := i
		refresh := refreshes[i]
		ewg.Go(func() error {
			if err := refresh(ctx, &dbState, cols); err != nil {
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

	return nil
}

func TestRefreshFormalDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, finalHeight abi.ChainEpoch) error {
	start := time.Now() //todo:test

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

	if err := RefreshBlockMsgs(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshBlockMsgsByMethodName(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshActorMsgsByMethodName(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshActorMsgs(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshActorTransferMsgs(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshMinedMsgsMaps(ctx, &dbState, cols); err != nil {
		return err
	}
	if err := RefreshTransfersForLargeAmount(ctx, &dbState, cols); err != nil {
		return err
	}

	if err := dbsm.Stm.SetDataBaseState(formal.Url(), dbState); err != nil {
		return err
	}

	//todo:test
	file, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/bell.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Errorf("open bell.txt failed: %v", err)
	}
	defer file.Close()
	_, err = io.WriteString(file, fmt.Sprintf("curtime: %v, startEpoch: %v, endEpoch: %v, NextEpochForBlockMsgsCount: %v, BlockMsgsCount: %v, elapsed: %v\n", fmt.Sprintf("%02d%02d%02d", time.Now().Day(), time.Now().Hour(), time.Now().Minute()), dbState.StartEpoch, dbState.EndEpoch, dbState.NextEpochForBlockMsgsCount, dbState.BlockMsgsCount, time.Now().Sub(start).String()))
	if err != nil {
		log.Errorf("write bell.txt failed: %v", err)
	}

	log.Infow("write to bell.txt successfully", "NextEpochForBlockMsgsCount", dbState.NextEpochForBlockMsgsCount)

	return nil
}

func refresh(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch, methodName, actorID string, f func(context.Context, *DataBaseState, Collections, *[]CountUtil, *abi.ChainEpoch, abi.ChainEpoch, string, string) error) ([]CountUtil, error) {
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

		err = f(ctx, dbState, cols, &countUtils, &tmpStartEpoch, curEpoch, methodName, actorID)
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

		err := f(ctx, tmpDBState, tmpCols, &countUtils, &tmpStartEpoch, curEpoch, methodName, actorID)
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

	countUtils, err := refresh(ctx, dbsm, curEpoch, methodName, "", refreshTotalCountForBlockMsgsByMethodName)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

//count_of_actormessages_by_methodname.js
func GetTotalCountForActorMsgByMethodName(ctx context.Context, actor, methodName string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForActorMsgByMethodName")

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

	countUtils, err := refresh(ctx, dbsm, curEpoch, methodName, actorID, refreshTotalCountForActorMsgByMethodName)
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
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		actorID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for actor %v", actor)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", actorID, refreshTotalCountForActorMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

// transfer_count_for_actor2.js
func GetTotalCountForActorTransferMsgs(ctx context.Context, actor string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
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
	case address.SECP256K1, address.BLS, address.Delegated:
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		actorID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for actor %v", actor)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", actorID, refreshTotalCountForActorTransferMsgs)
	if err != nil {
		return nil, err
	}

	return countUtils, nil
}

//出块列表 BlockHeader查，现在就是这样
func GetTotalCountForMinedMsgsMap(ctx context.Context, miner string, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) ([]CountUtil, error) {
	//clog := log.With("totalCount", "GetTotalCountForMinedMsgsMap")

	api := fullnode.API.GetAppropriateAPI()

	addr, err := address.NewFromString(buildnet.NetPrefix + miner)
	if err != nil {
		return nil, err
	}

	minerID := miner
	switch addr.Protocol() {
	case address.ID:
		minerID = addr.String()[1:]
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		minerID = ID.String()[1:]
	default:
		return nil, fmt.Errorf("unknow address type for miner %v", miner)
	}

	countUtils, err := refresh(ctx, dbsm, curEpoch, "", minerID, refreshTotalCountForMinedMsgsMap)
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

func GetAllBlockMsgsByMethodNameMap(ctx context.Context, dbsm *DataBaseStateManager, curEpoch abi.ChainEpoch) (map[string]int64, error) {
	blockMsgsByMethodNameMap := make(map[string]int64)

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

		for methodName, count := range dbState.BlockMsgsByMethodNameMap {
			blockMsgsByMethodNameMap[methodName] += count
		}

		if dbState.Formal {
			tmpStartEpoch = dbState.NextEpochForBlockMsgsByMethodName
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpDBState := DefaultDataBaseState(false, true, tmpStartEpoch, curEpoch+1)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		if err := RefreshBlockMsgsByMethodName(ctx, tmpDBState, tmpCols); err != nil {
			return nil, err
		}

		for methodName, count := range tmpDBState.BlockMsgsByMethodNameMap {
			blockMsgsByMethodNameMap[methodName] += count
		}
	}

	return blockMsgsByMethodNameMap, nil
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

		for methodName, actorMsgsCountMap := range dbState.ActorMsgsByMethodNameMap {
			if count, ok := actorMsgsCountMap[actorID]; ok {
				actorMsgsByMethodNameMap[methodName] += count
			}
		}

		if dbState.Formal {
			tmpStartEpoch = dbState.NextEpochForActorMsgsByMethodName
		}
	}

	// tmp每次都重新刷
	if !tmp.IsInvalidDB() {
		tmpDBState := DefaultDataBaseState(false, true, tmpStartEpoch, curEpoch+1)

		tmpCols, ok := dbsm.GetDBCollections(tmp.Url())
		if !ok {
			return nil, fmt.Errorf("url %v not found in DBCollectionsMap", tmp.Url())
		}

		if err := RefreshActorMsgsByMethodName(ctx, tmpDBState, tmpCols); err != nil {
			return nil, err
		}

		for methodName, actorMsgsCountMap := range tmpDBState.ActorMsgsByMethodNameMap {
			if count, ok := actorMsgsCountMap[actorID]; ok {
				actorMsgsByMethodNameMap[methodName] += count
			}
		}
	}

	return actorMsgsByMethodNameMap, nil
}

func refreshEpochRange(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
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

func refreshTotalCountForBlockMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForBlockMsgsCount), Count: dbState.BlockMsgsCount, Cols: cols})
		*tmpStartEpoch = dbState.NextEpochForBlockMsgsCount
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForBlockMsgsCount = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		if err := RefreshBlockMsgs(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForBlockMsgsCount), Count: dbState.BlockMsgsCount, Cols: cols})
		return nil
	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsCount, Cols: cols})
	return nil
}

func refreshTotalCountForBlockMsgsByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForBlockMsgsByMethodName), Count: dbState.BlockMsgsByMethodNameMap[methodName], Cols: cols})
		*tmpStartEpoch = dbState.NextEpochForBlockMsgsByMethodName
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForBlockMsgsByMethodName = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		if err := RefreshBlockMsgsByMethodName(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForBlockMsgsByMethodName), Count: dbState.BlockMsgsCount, Cols: cols})
		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsByMethodNameMap[methodName], Cols: cols})
	return nil
}

func refreshTotalCountForActorMsgByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		count := int64(0)
		if _, ok := dbState.ActorMsgsByMethodNameMap[methodName]; ok {
			count = dbState.ActorMsgsByMethodNameMap[methodName][actorID]
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorMsgsByMethodName), Count: count, Cols: cols})

		*tmpStartEpoch = dbState.NextEpochForBlockMsgsByMethodName
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForBlockMsgsByMethodName = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1
		dbState.ActorMsgsByMethodNameMap = make(map[string]map[string]int64)

		if err := RefreshActorMsgsByMethodName(ctx, dbState, cols); err != nil {
			return err
		}

		count := int64(0)
		if _, ok := dbState.ActorMsgsByMethodNameMap[methodName]; ok {
			count = dbState.ActorMsgsByMethodNameMap[methodName][actorID]
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorMsgsByMethodName), Count: count, Cols: cols})

		return nil

	}

	count := int64(0)
	if _, ok := dbState.ActorMsgsByMethodNameMap[methodName]; ok {
		count = dbState.ActorMsgsByMethodNameMap[methodName][actorID]
	}
	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols})

	return nil
}

func refreshTotalCountForActorMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorMsgsCount), Count: dbState.ActorMsgsCountMap[actorID], Cols: cols})

		*tmpStartEpoch = dbState.NextEpochForActorMsgsCount
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForActorMsgsCount = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		dbState.ActorMsgsCountMap = make(map[string]int64)
		if err := RefreshActorMsgs(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorMsgsCount), Count: dbState.ActorMsgsCountMap[actorID], Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorMsgsCountMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForActorTransferMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorTransfersCount), Count: dbState.ActorTransfersCountMap[actorID], Cols: cols})

		*tmpStartEpoch = dbState.NextEpochForActorTransfersCount
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForActorTransfersCount = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		dbState.ActorTransfersCountMap = make(map[string]int64)
		if err := RefreshActorTransferMsgs(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForActorTransfersCount), Count: dbState.ActorTransfersCountMap[actorID], Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorTransfersCountMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForMinedMsgsMap(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForMinedMsgs), Count: dbState.MinedMsgsMap[actorID], Cols: cols})

		*tmpStartEpoch = dbState.NextEpochForMinedMsgs
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForMinedMsgs = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		if err := RefreshMinedMsgsMaps(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForMinedMsgs), Count: dbState.MinedMsgsMap[actorID], Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.MinedMsgsMap[actorID], Cols: cols})

	return nil
}

func refreshTotalCountForTransfersForLargeAmount(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForTransfersLargeAmount), Count: dbState.TransfersLargeAmountCount, Cols: cols})

		*tmpStartEpoch = dbState.NextEpochForTransfersLargeAmount
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.NextEpochForMinedMsgs = *tmpStartEpoch, *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		if err := RefreshTransfersForLargeAmount(ctx, dbState, cols); err != nil {
			return err
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.NextEpochForTransfersLargeAmount), Count: dbState.TransfersLargeAmountCount, Cols: cols})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.TransfersLargeAmountCount, Cols: cols})

	return nil
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
func RefreshBlockMsgs(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "BlockMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForBlockMsgsCount, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh BlockMsgsCount successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForBlockMsgsCount {
		return nil
	}

	blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: ds.NextEpochForBlockMsgsCount}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: ds.EndEpoch}}}, {Key: "Depth", Value: 1}, {Key: "$or", Value: []bson.M{{"Msg.From": bson.D{{Key: "$regex", Value: "^1"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^3"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^4"}}}}}}

	var (
		count int64
		err   error
	)
	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err = col.CountDocuments(ctx, blockFilter)
			if err != nil {
				return err
			}

			previousCount := ds.BlockMsgsCount
			ds.BlockMsgsCount = previousCount + count
			ds.NextEpochForBlockMsgsCount = ds.EndEpoch
		}
	}

	log.Infow("RefreshBlockMsgs", "count", count)

	return nil
}

func RefreshBlockMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "BlockMsgsByMethodName")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForBlockMsgsByMethodName, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh BlockMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForBlockMsgsByMethodName {
		return nil
	}

	var countOfBlockMessageByMethodNames []model.CountOfBlockMessageByMethodName
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForBlockMsgsByMethodName), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetCountOfBlockMessagesByMethodNameAggregator()))
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &countOfBlockMessageByMethodNames)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			for _, countOfBlockMessageByMethodName := range countOfBlockMessageByMethodNames {
				ds.BlockMsgsByMethodNameMap[countOfBlockMessageByMethodName.MethodName] += countOfBlockMessageByMethodName.Count
			}

			ds.NextEpochForBlockMsgsByMethodName = ds.EndEpoch
		}
	}

	// log.Warnf

	return nil
}

func RefreshActorMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "ActorMsgsByMethodName")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForActorMsgsByMethodName, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForActorMsgsByMethodName {
		return nil
	}

	var countOfActorMessagesByMethodNames []model.CountOfActorMessagesByMethodName
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForActorMsgsByMethodName), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetCountOfActorMessagesByMethodNameAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe) //, options.Aggregate().SetAllowDiskUse(true)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &countOfActorMessagesByMethodNames)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			if len(countOfActorMessagesByMethodNames) == 0 {
				rlog.Warnf("no actor between %v and %v", ds.NextEpochForActorMsgsByMethodName, ds.EndEpoch-1)
				ds.NextEpochForActorMsgsByMethodName = ds.EndEpoch
				return nil
			}

			actorMsgsByMethodNameMap := make(map[string]map[string]int64)

			api := fullnode.API.GetAppropriateAPI()

			for _, countOfActorMessagesByMethodName := range countOfActorMessagesByMethodNames {
				methodName, from, to := countOfActorMessagesByMethodName.Method, countOfActorMessagesByMethodName.From, countOfActorMessagesByMethodName.To
				fromID, toID, fromErr, toErr := GetIDForAddr(ctx, from, to, rlog, api)
				if !fromErr && !toErr {
					if fromID == toID {
						if _, ok := actorMsgsByMethodNameMap[methodName]; !ok {
							actorMsgsByMethodNameMap[methodName] = make(map[string]int64)
							actorMsgsByMethodNameMap[methodName][fromID]++
						} else {
							actorMsgsByMethodNameMap[methodName][fromID]++
						}
					} else {
						if _, ok := actorMsgsByMethodNameMap[methodName]; !ok {
							actorMsgsByMethodNameMap[methodName] = make(map[string]int64)
							actorMsgsByMethodNameMap[methodName][fromID]++
						} else {
							actorMsgsByMethodNameMap[methodName][fromID]++
						}

						if _, ok := actorMsgsByMethodNameMap[methodName]; !ok {
							actorMsgsByMethodNameMap[methodName] = make(map[string]int64)
							actorMsgsByMethodNameMap[methodName][toID]++
						} else {
							actorMsgsByMethodNameMap[methodName][toID]++
						}
					}
				} else if !fromErr {
					if _, ok := actorMsgsByMethodNameMap[methodName]; !ok {
						actorMsgsByMethodNameMap[methodName] = make(map[string]int64)
						actorMsgsByMethodNameMap[methodName][fromID]++
					} else {
						actorMsgsByMethodNameMap[methodName][fromID]++
					}
				} else if !toErr {
					if _, ok := actorMsgsByMethodNameMap[methodName]; !ok {
						actorMsgsByMethodNameMap[methodName] = make(map[string]int64)
						actorMsgsByMethodNameMap[methodName][toID]++
					} else {
						actorMsgsByMethodNameMap[methodName][toID]++
					}
				}
			}

			for methodName, actorMsgs := range actorMsgsByMethodNameMap {
				for actorID, count := range actorMsgs {
					if _, ok := ds.ActorMsgsByMethodNameMap[methodName]; !ok {
						ds.ActorMsgsByMethodNameMap[methodName] = make(map[string]int64)
						ds.ActorMsgsByMethodNameMap[methodName][actorID] += count
					} else {
						ds.ActorMsgsByMethodNameMap[methodName][actorID] += count
					}
				}
			}
			ds.NextEpochForActorMsgsByMethodName = ds.EndEpoch
		}
	}

	return nil
}

func RefreshActorMsgs(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "ActorMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForActorMsgsCount, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorMsgsCountMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForActorMsgsCount {
		return nil
	}

	var allActorsRes []model.AllActorsRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForActorMsgsCount), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetAllActorsForBlockMessageAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe, options.Aggregate().SetAllowDiskUse(true)) // todo
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &allActorsRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			if len(allActorsRes) == 0 {
				rlog.Warnf("no actor between %v and %v", ds.NextEpochForActorMsgsCount, ds.EndEpoch-1)
				ds.NextEpochForActorMsgsCount = ds.EndEpoch
				return nil
			}

			actorMsgsMap := make(map[string]int64)
			api := fullnode.API.GetAppropriateAPI()
			for i := range allActorsRes {
				from, to := allActorsRes[i].From, allActorsRes[i].To
				fromID, toID, fromErr, toErr := GetIDForAddr(ctx, from, to, rlog, api)

				if !fromErr && !toErr {
					if fromID == toID {
						actorMsgsMap[fromID]++
					} else {
						actorMsgsMap[fromID]++
						actorMsgsMap[toID]++
					}
				} else if !fromErr {
					actorMsgsMap[fromID]++
				} else if !toErr {
					actorMsgsMap[toID]++
				}
			}

			for actor, count := range actorMsgsMap {
				ds.ActorMsgsCountMap[actor] += count
			}
			ds.NextEpochForActorMsgsCount = ds.EndEpoch
		}
	}

	return nil
}

func RefreshActorTransferMsgs(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "ActorTransferMsgs")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForActorTransfersCount, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh ActorTransfersCountMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForActorTransfersCount {
		return nil
	}

	var allActorsRes []model.AllActorsRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForActorTransfersCount), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetCountOfTransfersForActor2Aggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe, options.Aggregate().SetAllowDiskUse(true)) // todo
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &allActorsRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			if len(allActorsRes) == 0 {
				rlog.Warnf("no actor between %v and %v", ds.NextEpochForActorTransfersCount, ds.EndEpoch-1)
				ds.NextEpochForActorTransfersCount = ds.EndEpoch
				return nil
			}

			actorMsgsMap := make(map[string]int64)
			api := fullnode.API.GetAppropriateAPI()
			for i := range allActorsRes {
				from, to := allActorsRes[i].From, allActorsRes[i].To
				fromID, toID, fromErr, toErr := GetIDForAddr(ctx, from, to, rlog, api)

				if !fromErr && !toErr {
					if fromID == toID {
						actorMsgsMap[fromID]++
					} else {
						actorMsgsMap[fromID]++
						actorMsgsMap[toID]++
					}
				} else if !fromErr {
					actorMsgsMap[fromID]++
				} else if !toErr {
					actorMsgsMap[toID]++
				}
			}

			for actor, count := range actorMsgsMap {
				ds.ActorTransfersCountMap[actor] += count
			}
			ds.NextEpochForActorTransfersCount = ds.EndEpoch
		}
	}

	return nil
}

func RefreshMinedMsgsMaps(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "MinedMsgsMap")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForMinedMsgs, ds.EndEpoch-1
	defer func() {
		rlog.Infof("refresh MinedMsgsMap successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForMinedMsgs {
		return nil
	}

	var minedCountForMinersRes []model.MinedCountForMinersRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForMinedMsgs), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetMinedCountForMinersAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "BlockHeader"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &minedCountForMinersRes)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			if len(minedCountForMinersRes) == 0 {
				rlog.Warnf("no actor between %v and %v", ds.NextEpochForActorMsgsCount, ds.EndEpoch-1)
				ds.NextEpochForMinedMsgs = ds.EndEpoch
				return nil
			}

			minedMsgsMap := make(map[string]int64)
			for _, minedCountForMiners := range minedCountForMinersRes {
				miner := minedCountForMiners.Miner
				api := fullnode.API.GetAppropriateAPI()

				minerErr := false
				var minerID address.Address
				// todo: 消息中有无效的地址暂时跳过不存储
				mAddr, err := address.NewFromString(buildnet.NetPrefix + miner)
				if err != nil {
					rlog.Errorf("invalid miner address: %v", miner)
					minerErr = true
				}

				if !minerErr {
					minerID, err = api.StateLookupID(ctx, mAddr, types.EmptyTSK)
					if err != nil {
						rlog.Errorf("lookup ID for miner address %v failed: %v", miner, err)
						minerErr = true
					}
				}

				if !minerErr {
					minedMsgsMap[minerID.String()[1:]] = minedCountForMiners.MinedCount
				}
			}

			for miner, count := range minedMsgsMap {
				ds.MinedMsgsMap[miner] += count
			}
			ds.NextEpochForMinedMsgs = ds.EndEpoch
		}
	}

	return nil
}

func RefreshTransfersForLargeAmount(ctx context.Context, ds *DataBaseState, cols Collections) error {
	rlog := log.With("refresh", "TransfersForLargeAmount")

	start := time.Now()
	startEpoch, endEpoch := ds.NextEpochForTransfersLargeAmount, ds.EndEpoch-1
	defer func() {

		rlog.Infof("refresh TransfersLargeAmountCount successfully between %v and %v, elapsed: %v", startEpoch, endEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= ds.NextEpochForTransfersLargeAmount {
		return nil
	}

	var countOfLargeAmountTransfers []model.CountOfLargeAmountTransfers
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(ds.NextEpochForTransfersLargeAmount), EndEpoch: int64(ds.EndEpoch)}, string(monitor.GetCountOfLargeAmountTransfersAggregator())) // todo: ExecTrace
	if err != nil {
		//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
		return err
	}

	tableName := "ExecTrace"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				//log.Errorf("get all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			err = cur.All(ctx, &countOfLargeAmountTransfers)
			if err != nil {
				//log.Errorf("cur.All for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
				return err
			}

			if len(countOfLargeAmountTransfers) == 0 {
				rlog.Warnf("no large amount transfers between %v and %v", ds.NextEpochForTransfersLargeAmount, ds.EndEpoch-1)
				ds.NextEpochForTransfersLargeAmount = ds.EndEpoch
				return nil
			}

			ds.TransfersLargeAmountCount += countOfLargeAmountTransfers[0].Count
			ds.NextEpochForTransfersLargeAmount = ds.EndEpoch
		}
	}

	return nil
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
