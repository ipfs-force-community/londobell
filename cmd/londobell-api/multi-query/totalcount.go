package multiquery

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hashicorp/go-multierror"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	tmpLimit            = 16
	tmpInterval         = 2880 * 7
)

var (
	refreshOnce sync.Once
	Refreshes   = make([]func(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error, 0)
)

func init() {
	refreshOnce.Do(func() {
		Refreshes = append(Refreshes, RefreshBlockMsgs, RefreshBlockMsgsByMethodName, RefreshActorMsgsByMethodName, RefreshActorMsgs, RefreshActorTransferMsgs, RefreshMinedMsgsMaps, RefreshTransfersForLargeAmount)
	})
}

func PeriodicRefreshDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, limit, interval int) {
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
			log.Infof("begin PeriodicRefreshDataBaseState for formal, finalHeight: %v", res[0].Epoch)
			if err := RefreshFormalDataBaseState(ctx, dbsm, res[0].Epoch, limit, interval); err != nil {
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
func RefreshFormalDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, finalHeight abi.ChainEpoch, limit, interval int) error {
	formal := dbsm.GetFormalCfg()

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
		return nil
	}

	dbState, found, err := dbsm.Seg.Find(context.TODO(), formal.Url())
	//dbState, found, err := dbsm.Stm.LoadDataBaseState(formal.Url())
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

	for dbState.EndEpoch <= finalHeight+1 {
		dbState.EndEpoch = dbState.EndEpoch + abi.ChainEpoch(limit*interval)
		if dbState.EndEpoch > finalHeight+1 {
			dbState.EndEpoch = finalHeight + 1
		}

		var ewg multierror.Group
		for i := range Refreshes {
			i := i
			refresh := Refreshes[i]
			ewg.Go(func() error {
				if err := refresh(ctx, &dbState, cols, limit, interval); err != nil {
					return err
				}

				return nil
			})
		}

		if err := ewg.Wait(); err != nil {
			log.Errorf("RefreshFormalDataBaseState failed: %v", err)
			return err
		}

		// 写入数据库
		updated, err := dbsm.Seg.Update(ctx, formal.Url(), dbState)
		if err != nil {
			return err
		}

		//if err := dbsm.Stm.SetDataBaseState(formal.Url(), dbState); err != nil {
		//	return err
		//}

		dbsm.DBStateCache.SetDataBase(formal.Url(), &dbState)

		log.Infof("RefreshFormalDataBaseState successfully for part, dbState.EndEpoch: %v, updated: %v", dbState.EndEpoch, updated)
	}

	log.Infof("RefreshFormalDataBaseState successfully, dbState.EndEpoch: %v", dbState.EndEpoch)

	return nil
}

func TestRefreshFormalDataBaseState(ctx context.Context, dbsm *DataBaseStateManager, finalHeight abi.ChainEpoch) error {
	start := time.Now() //todo:test

	formal := dbsm.GetFormalCfg()

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
		return nil
	}

	dbState, found, err := dbsm.Seg.Find(context.TODO(), formal.Url())
	//dbState, found, err := dbsm.Stm.LoadDataBaseState(formal.Url())
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

	if err := RefreshBlockMsgs(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshBlockMsgsByMethodName(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshActorMsgsByMethodName(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshActorMsgs(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshActorTransferMsgs(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshMinedMsgsMaps(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}
	if err := RefreshTransfersForLargeAmount(ctx, &dbState, cols, tmpLimit, tmpInterval); err != nil {
		return err
	}

	_, err = dbsm.Seg.Update(ctx, formal.Url(), dbState)
	//err := dbsm.Stm.SetDataBaseState(formal.Url(), dbState)
	if err != nil {
		return err
	}

	//todo:test
	file, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/bell.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		log.Errorf("open bell.txt failed: %v", err)
	}
	defer file.Close()
	_, err = io.WriteString(file, fmt.Sprintf("curtime: %v, startEpoch: %v, endEpoch: %v, elapsed: %v\n", fmt.Sprintf("%02d%02d%02d", time.Now().Day(), time.Now().Hour(), time.Now().Minute()), dbState.StartEpoch, dbState.EndEpoch, time.Now().Sub(start).String()))
	if err != nil {
		log.Errorf("write bell.txt failed: %v", err)
	}

	log.Infow("write to bell.txt successfully", "endEpoch", dbState.EndEpoch)

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

		for i := range dbState.BlockMsgsByMethodNameStates {
			for methodName, count := range dbState.BlockMsgsByMethodNameStates[i].BlockMsgsByMethodNameMap {
				blockMsgsByMethodNameMap[methodName] += count
			}
		}

		sort.Slice(dbState.BlockMsgsByMethodNameStates, func(i, j int) bool {
			return dbState.BlockMsgsByMethodNameStates[i].StartEpoch > dbState.BlockMsgsByMethodNameStates[j].StartEpoch
		})

		if dbState.Formal {
			tmpStartEpoch = dbState.StartEpoch
			if len(dbState.BlockMsgsByMethodNameStates) > 0 {
				tmpStartEpoch = dbState.BlockMsgsByMethodNameStates[0].EndEpoch
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

		if err := RefreshBlockMsgsByMethodName(ctx, tmpDBState, tmpCols, tmpLimit, tmpInterval); err != nil {
			return nil, err
		}

		for methodName, count := range tmpDBState.BlockMsgsByMethodNameStates[0].BlockMsgsByMethodNameMap {
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

		for i := range dbState.ActorMsgsByMethodNameStates {
			for methodName, actorMsgsCountMap := range dbState.ActorMsgsByMethodNameStates[i].ActorMsgsByMethodNameMap {
				if count, ok := actorMsgsCountMap[actorID]; ok {
					actorMsgsByMethodNameMap[methodName] += count
				}
			}
		}

		sort.Slice(dbState.ActorMsgsByMethodNameStates, func(i, j int) bool {
			return dbState.ActorMsgsByMethodNameStates[i].StartEpoch > dbState.ActorMsgsByMethodNameStates[j].StartEpoch
		})

		if dbState.Formal {
			tmpStartEpoch = dbState.StartEpoch
			if len(dbState.ActorMsgsByMethodNameStates) > 0 {
				tmpStartEpoch = dbState.ActorMsgsByMethodNameStates[0].EndEpoch
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

		if err := RefreshActorMsgsByMethodName(ctx, tmpDBState, tmpCols, tmpLimit, tmpInterval); err != nil {
			return nil, err
		}

		for methodName, actorMsgsCountMap := range tmpDBState.ActorMsgsByMethodNameStates[0].ActorMsgsByMethodNameMap {
			if count, ok := actorMsgsCountMap[actorID]; ok {
				actorMsgsByMethodNameMap[methodName] += count
			}
		}
	}

	return actorMsgsByMethodNameMap, nil
}

func refreshEpochRange(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	innerStates := make([]InnerState, 0)
	for i := range dbState.BlockMsgsStates {
		innerStates = append(innerStates, InnerState{Start: int64(dbState.BlockMsgsStates[i].StartEpoch), End: int64(dbState.BlockMsgsStates[i].EndEpoch)})
	}

	if dbState.Formal {
		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols, InnerStates: innerStates})
		*tmpStartEpoch = dbState.EndEpoch
		return nil
	}

	if dbState.Tmp {
		// tmp就不内部分区了
		dbState.StartEpoch = *tmpStartEpoch
		dbState.EndEpoch = curEpoch + 1

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch)})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols, InnerStates: innerStates})
		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Cols: cols, InnerStates: innerStates})
	return nil
}

func refreshTotalCountForBlockMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.BlockMsgsStates {
		totalCount += dbState.BlockMsgsStates[i].BlockMsgsCount
		count = dbState.BlockMsgsStates[i].BlockMsgsCount
		innerStates = append(innerStates, InnerState{Start: int64(dbState.BlockMsgsStates[i].StartEpoch), End: int64(dbState.BlockMsgsStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.BlockMsgsStates, func(i, j int) bool {
			return dbState.BlockMsgsStates[i].StartEpoch > dbState.BlockMsgsStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.BlockMsgsStates) > 0 {
			endEpoch = dbState.BlockMsgsStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})
		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshBlockMsgs(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.BlockMsgsStates[0].StartEpoch), End: int64(dbState.BlockMsgsStates[0].EndEpoch), Count: dbState.BlockMsgsStates[0].BlockMsgsCount})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsStates[0].BlockMsgsCount, Cols: cols, InnerStates: innerStates})
		return nil
	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})
	return nil
}

func refreshTotalCountForBlockMsgsByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.BlockMsgsByMethodNameStates {
		totalCount += dbState.BlockMsgsByMethodNameStates[i].BlockMsgsByMethodNameMap[methodName]
		count = dbState.BlockMsgsByMethodNameStates[i].BlockMsgsByMethodNameMap[methodName]
		innerStates = append(innerStates, InnerState{Start: int64(dbState.BlockMsgsByMethodNameStates[i].StartEpoch), End: int64(dbState.BlockMsgsByMethodNameStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.BlockMsgsByMethodNameStates, func(i, j int) bool {
			return dbState.BlockMsgsByMethodNameStates[i].StartEpoch > dbState.BlockMsgsByMethodNameStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.BlockMsgsByMethodNameStates) > 0 {
			endEpoch = dbState.BlockMsgsByMethodNameStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})
		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshBlockMsgsByMethodName(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.BlockMsgsByMethodNameStates[0].StartEpoch), End: int64(dbState.BlockMsgsByMethodNameStates[0].EndEpoch), Count: dbState.BlockMsgsByMethodNameStates[0].BlockMsgsByMethodNameMap[methodName]})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.BlockMsgsByMethodNameStates[0].BlockMsgsByMethodNameMap[methodName], Cols: cols, InnerStates: innerStates})
		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})
	return nil
}

func refreshTotalCountForActorMsgByMethodName(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.ActorMsgsByMethodNameStates {
		if _, ok := dbState.ActorMsgsByMethodNameStates[i].ActorMsgsByMethodNameMap[methodName]; ok {
			totalCount += dbState.ActorMsgsByMethodNameStates[i].ActorMsgsByMethodNameMap[methodName][actorID]
			count = dbState.ActorMsgsByMethodNameStates[i].ActorMsgsByMethodNameMap[methodName][actorID]
		}

		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorMsgsByMethodNameStates[i].StartEpoch), End: int64(dbState.ActorMsgsByMethodNameStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.ActorMsgsByMethodNameStates, func(i, j int) bool {
			return dbState.ActorMsgsByMethodNameStates[i].StartEpoch > dbState.ActorMsgsByMethodNameStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.ActorMsgsByMethodNameStates) > 0 {
			endEpoch = dbState.ActorMsgsByMethodNameStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshActorMsgsByMethodName(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		count := int64(0)
		if _, ok := dbState.ActorMsgsByMethodNameStates[0].ActorMsgsByMethodNameMap[methodName]; ok {
			count = dbState.ActorMsgsByMethodNameStates[0].ActorMsgsByMethodNameMap[methodName][actorID]
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorMsgsByMethodNameStates[0].StartEpoch), End: int64(dbState.ActorMsgsByMethodNameStates[0].EndEpoch), Count: count})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: count, Cols: cols, InnerStates: innerStates})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

	return nil
}

func refreshTotalCountForActorMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.ActorMsgsCountStates {
		totalCount += dbState.ActorMsgsCountStates[i].ActorMsgsCountMap[actorID]
		count = dbState.ActorMsgsCountStates[i].ActorMsgsCountMap[actorID]
		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorMsgsCountStates[i].StartEpoch), End: int64(dbState.ActorMsgsCountStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.ActorMsgsCountStates, func(i, j int) bool {
			return dbState.ActorMsgsCountStates[i].StartEpoch > dbState.ActorMsgsCountStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.ActorMsgsCountStates) > 0 {
			endEpoch = dbState.ActorMsgsCountStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshActorMsgs(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorMsgsCountStates[0].StartEpoch), End: int64(dbState.ActorMsgsCountStates[0].EndEpoch), Count: dbState.ActorMsgsCountStates[0].ActorMsgsCountMap[actorID]})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorMsgsCountStates[0].ActorMsgsCountMap[actorID], Cols: cols, InnerStates: innerStates})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

	return nil
}

func refreshTotalCountForActorTransferMsgs(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.ActorTransfersCountStates {
		totalCount += dbState.ActorTransfersCountStates[i].ActorTransfersCountMap[actorID]
		count = dbState.ActorTransfersCountStates[i].ActorTransfersCountMap[actorID]
		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorTransfersCountStates[i].StartEpoch), End: int64(dbState.ActorTransfersCountStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.ActorTransfersCountStates, func(i, j int) bool {
			return dbState.ActorTransfersCountStates[i].StartEpoch > dbState.ActorTransfersCountStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.ActorTransfersCountStates) > 0 {
			endEpoch = dbState.ActorTransfersCountStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshActorTransferMsgs(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.ActorTransfersCountStates[0].StartEpoch), End: int64(dbState.ActorTransfersCountStates[0].EndEpoch), Count: dbState.ActorTransfersCountStates[0].ActorTransfersCountMap[actorID]})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.ActorTransfersCountStates[0].ActorTransfersCountMap[actorID], Cols: cols, InnerStates: innerStates})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

	return nil
}

func refreshTotalCountForMinedMsgsMap(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.MinedMsgsStates {
		totalCount += dbState.MinedMsgsStates[i].MinedMsgsMap[actorID]
		count = dbState.MinedMsgsStates[i].MinedMsgsMap[actorID]
		innerStates = append(innerStates, InnerState{Start: int64(dbState.MinedMsgsStates[i].StartEpoch), End: int64(dbState.MinedMsgsStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.MinedMsgsStates, func(i, j int) bool {
			return dbState.MinedMsgsStates[i].StartEpoch > dbState.MinedMsgsStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.MinedMsgsStates) > 0 {
			endEpoch = dbState.MinedMsgsStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshMinedMsgsMaps(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.MinedMsgsStates[0].StartEpoch), End: int64(dbState.MinedMsgsStates[0].EndEpoch), Count: dbState.MinedMsgsStates[0].MinedMsgsMap[actorID]})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.MinedMsgsStates[0].MinedMsgsMap[actorID], Cols: cols, InnerStates: innerStates})

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

	return nil
}

func refreshTotalCountForTransfersForLargeAmount(ctx context.Context, dbState *DataBaseState, cols Collections, countUtils *[]CountUtil, tmpStartEpoch *abi.ChainEpoch, curEpoch abi.ChainEpoch, methodName, actorID string) error {
	totalCount, count := int64(0), int64(0)
	innerStates := make([]InnerState, 0)
	for i := range dbState.TransfersLargeAmountStates {
		totalCount += dbState.TransfersLargeAmountStates[i].TransfersLargeAmountCount
		count = dbState.TransfersLargeAmountStates[i].TransfersLargeAmountCount
		innerStates = append(innerStates, InnerState{Start: int64(dbState.TransfersLargeAmountStates[i].StartEpoch), End: int64(dbState.TransfersLargeAmountStates[i].EndEpoch), Count: count})
	}

	if dbState.Formal {
		sort.Slice(dbState.TransfersLargeAmountStates, func(i, j int) bool {
			return dbState.TransfersLargeAmountStates[i].StartEpoch > dbState.TransfersLargeAmountStates[j].StartEpoch
		})

		endEpoch := dbState.StartEpoch
		if len(dbState.TransfersLargeAmountStates) > 0 {
			endEpoch = dbState.TransfersLargeAmountStates[0].EndEpoch
		}

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(endEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

		*tmpStartEpoch = endEpoch
		return nil
	}

	if dbState.Tmp {
		dbState.StartEpoch, dbState.EndEpoch = *tmpStartEpoch, curEpoch+1

		if err := RefreshTransfersForLargeAmount(ctx, dbState, cols, tmpLimit, tmpInterval); err != nil {
			return err
		}

		innerStates := make([]InnerState, 0)
		innerStates = append(innerStates, InnerState{Start: int64(dbState.TransfersLargeAmountStates[0].StartEpoch), End: int64(dbState.TransfersLargeAmountStates[0].EndEpoch), Count: dbState.TransfersLargeAmountStates[0].TransfersLargeAmountCount})

		*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: dbState.TransfersLargeAmountStates[0].TransfersLargeAmountCount, Cols: cols, InnerStates: innerStates}) // todo:nil

		return nil

	}

	*countUtils = append(*countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch), Count: totalCount, Cols: cols, InnerStates: innerStates})

	return nil
}

type refreshType struct {
	startEpoch abi.ChainEpoch
	endEpoch   abi.ChainEpoch

	latestStartEpoch abi.ChainEpoch
	previousState    interface{}
	storeType        string
}

func getRefreshLists(latestStartEpoch, latestEndEpoch, dsEndEpoch abi.ChainEpoch, previousState interface{}, interval int) []refreshType {
	refreshLists := make([]refreshType, 0)
	for latestEndEpoch < dsEndEpoch {
		if /*latestStartEpoch == latestEndEpoch ||*/ latestEndEpoch-latestStartEpoch >= abi.ChainEpoch(interval) {
			// new next
			startEpoch := latestEndEpoch
			endEpoch := latestEndEpoch + abi.ChainEpoch(interval)
			if endEpoch > dsEndEpoch {
				endEpoch = dsEndEpoch
			}

			// agg [startEpoch,endEpoch)

			// store new [startEpoch,endEpoch)
			refreshLists = append(refreshLists, refreshType{startEpoch: startEpoch, endEpoch: endEpoch, storeType: "new"})

			latestStartEpoch, latestEndEpoch = startEpoch, endEpoch
		} else if latestEndEpoch-latestStartEpoch < abi.ChainEpoch(interval) {
			// 每次应该最多只执行一次
			startEpoch := latestEndEpoch
			endEpoch := latestStartEpoch + abi.ChainEpoch(interval)
			if endEpoch > dsEndEpoch {
				endEpoch = dsEndEpoch
			}

			// agg [startEpoch,endEpoch)

			// store add [latestStartEpoch,endEpoch)
			refreshLists = append(refreshLists, refreshType{startEpoch: startEpoch, endEpoch: endEpoch, latestStartEpoch: latestStartEpoch, previousState: previousState, storeType: "add"})

			latestEndEpoch = endEpoch
		}
	}

	return refreshLists
}

// 更新formal，新增cold   异步程序更新cold state,   中间会有一段时间数据断层？
func RefreshBlockMsgs(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "BlockMsgs")

	start := time.Now()

	sort.Slice(ds.BlockMsgsStates, func(i, j int) bool {
		return ds.BlockMsgsStates[i].StartEpoch < ds.BlockMsgsStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.BlockMsgsStates) - 1
	if len(ds.BlockMsgsStates) > 0 {
		latestStartEpoch = ds.BlockMsgsStates[previousLength].StartEpoch
		latestEndEpoch = ds.BlockMsgsStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh BlockMsgsCount successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.BlockMsgsStates[previousLength].BlockMsgsCount, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				// agg [startEpoch,endEpoch)
				blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: refreshList.startEpoch}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: refreshList.endEpoch}}}, {Key: "Depth", Value: 1}, {Key: "$or", Value: []bson.M{{"Msg.From": bson.D{{Key: "$regex", Value: "^1"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^3"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^4"}}}}}}

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

						// store new [startEpoch,endEpoch)
						mutex.Lock()
						ds.BlockMsgsStates = append(ds.BlockMsgsStates, BlockMsgsState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, BlockMsgsCount: count})
						mutex.Unlock()
					}
				}
			} else if refreshList.storeType == "add" {
				// agg [startEpoch,endEpoch)
				blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: refreshList.startEpoch}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: refreshList.endEpoch}}}, {Key: "Depth", Value: 1}, {Key: "$or", Value: []bson.M{{"Msg.From": bson.D{{Key: "$regex", Value: "^1"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^3"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^4"}}}}}}

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

						// store add [latestStartEpoch,endEpoch)
						blockMsgsCount := refreshList.previousState.(int64) + count

						mutex.Lock()
						ds.BlockMsgsStates[previousLength] = BlockMsgsState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, BlockMsgsCount: blockMsgsCount}
						mutex.Unlock()
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	return nil
}

func RefreshBlockMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "BlockMsgsByMethodName")

	start := time.Now()

	sort.Slice(ds.BlockMsgsByMethodNameStates, func(i, j int) bool {
		return ds.BlockMsgsByMethodNameStates[i].StartEpoch < ds.BlockMsgsByMethodNameStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.BlockMsgsByMethodNameStates) - 1
	if len(ds.BlockMsgsByMethodNameStates) > 0 {
		latestStartEpoch = ds.BlockMsgsByMethodNameStates[previousLength].StartEpoch
		latestEndEpoch = ds.BlockMsgsByMethodNameStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh BlockMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.BlockMsgsByMethodNameStates[previousLength].BlockMsgsByMethodNameMap, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var countOfBlockMessageByMethodNames []model.CountOfBlockMessageByMethodName
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfBlockMessagesByMethodNameAggregator()))
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store new [startEpoch,endEpoch)
						blockMsgsByMethodNameMap := make(map[string]int64)
						for _, countOfBlockMessageByMethodName := range countOfBlockMessageByMethodNames {
							blockMsgsByMethodNameMap[countOfBlockMessageByMethodName.MethodName] += countOfBlockMessageByMethodName.Count
						}

						mutex.Lock()
						ds.BlockMsgsByMethodNameStates = append(ds.BlockMsgsByMethodNameStates, BlockMsgsByMethodNameState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, BlockMsgsByMethodNameMap: blockMsgsByMethodNameMap})
						mutex.Unlock()
					}
				}
			} else if refreshList.storeType == "add" {
				var countOfBlockMessageByMethodNames []model.CountOfBlockMessageByMethodName
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfBlockMessagesByMethodNameAggregator()))
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store add [latestStartEpoch,endEpoch)
						blockMsgsByMethodNameMap := refreshList.previousState.(map[string]int64)
						for _, countOfBlockMessageByMethodName := range countOfBlockMessageByMethodNames {
							blockMsgsByMethodNameMap[countOfBlockMessageByMethodName.MethodName] += countOfBlockMessageByMethodName.Count
						}

						ds.BlockMsgsByMethodNameStates[previousLength] = BlockMsgsByMethodNameState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, BlockMsgsByMethodNameMap: blockMsgsByMethodNameMap}
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	// log.Warnf

	return nil
}

func RefreshActorMsgsByMethodName(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "ActorMsgsByMethodName")

	start := time.Now()

	sort.Slice(ds.ActorMsgsByMethodNameStates, func(i, j int) bool {
		return ds.ActorMsgsByMethodNameStates[i].StartEpoch < ds.ActorMsgsByMethodNameStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.ActorMsgsByMethodNameStates) - 1
	if len(ds.ActorMsgsByMethodNameStates) > 0 {
		latestStartEpoch = ds.ActorMsgsByMethodNameStates[previousLength].StartEpoch
		latestEndEpoch = ds.ActorMsgsByMethodNameStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh ActorMsgsByMethodNameMap successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.ActorMsgsByMethodNameStates[previousLength].ActorMsgsByMethodNameMap, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var countOfActorMessagesByMethodNames []model.CountOfActorMessagesByMethodName
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfActorMessagesByMethodNameAggregator())) // todo: ExecTrace
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

						// store new [startEpoch,endEpoch)
						newActorMsgsByMethodNameMap := make(map[string]map[string]int64)
						if len(countOfActorMessagesByMethodNames) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorMsgsByMethodNameStates = append(ds.ActorMsgsByMethodNameStates, ActorMsgsByMethodNameState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsByMethodNameMap: newActorMsgsByMethodNameMap})
							mutex.Unlock()
						} else {
							api := fullnode.API.GetAppropriateAPI()
							actorMsgsByMethodNameMap := make(map[string]map[string]int64)

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
									if _, ok := newActorMsgsByMethodNameMap[methodName]; !ok {
										newActorMsgsByMethodNameMap[methodName] = make(map[string]int64)
										newActorMsgsByMethodNameMap[methodName][actorID] += count
									} else {
										newActorMsgsByMethodNameMap[methodName][actorID] += count
									}
								}
							}

							mutex.Lock()
							ds.ActorMsgsByMethodNameStates = append(ds.ActorMsgsByMethodNameStates, ActorMsgsByMethodNameState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsByMethodNameMap: newActorMsgsByMethodNameMap})
							mutex.Unlock()
						}
					}
				}
			} else if refreshList.storeType == "add" {
				var countOfActorMessagesByMethodNames []model.CountOfActorMessagesByMethodName
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfActorMessagesByMethodNameAggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						newActorMsgsByMethodNameMap := refreshList.previousState.(map[string]map[string]int64)

						if len(countOfActorMessagesByMethodNames) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorMsgsByMethodNameStates[len(ds.ActorMsgsByMethodNameStates)-1] = ActorMsgsByMethodNameState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsByMethodNameMap: newActorMsgsByMethodNameMap}
							mutex.Unlock()
						} else {
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
									if _, ok := newActorMsgsByMethodNameMap[methodName]; !ok {
										newActorMsgsByMethodNameMap[methodName] = make(map[string]int64)
										newActorMsgsByMethodNameMap[methodName][actorID] += count
									} else {
										newActorMsgsByMethodNameMap[methodName][actorID] += count
									}
								}
							}

							mutex.Lock()
							ds.ActorMsgsByMethodNameStates[previousLength] = ActorMsgsByMethodNameState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsByMethodNameMap: newActorMsgsByMethodNameMap}
							mutex.Unlock()

						}
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	return nil
}

func RefreshActorMsgs(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "ActorMsgs")

	start := time.Now()

	sort.Slice(ds.ActorMsgsCountStates, func(i, j int) bool {
		return ds.ActorMsgsCountStates[i].StartEpoch < ds.ActorMsgsCountStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.ActorMsgsCountStates) - 1
	if len(ds.ActorMsgsCountStates) > 0 {
		latestStartEpoch = ds.ActorMsgsCountStates[previousLength].StartEpoch
		latestEndEpoch = ds.ActorMsgsCountStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh ActorMsgsCountMap successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.ActorMsgsCountStates[previousLength].ActorMsgsCountMap, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var allActorsRes []model.AllActorsRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetAllActorsForBlockMessageAggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store new [startEpoch,endEpoch)
						actorMsgCountMap := make(map[string]int64)

						if len(allActorsRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorMsgsCountStates = append(ds.ActorMsgsCountStates, ActorMsgsCountState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsCountMap: actorMsgCountMap})
							mutex.Unlock()
						} else {
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
								actorMsgCountMap[actor] += count
							}

							mutex.Lock()
							ds.ActorMsgsCountStates = append(ds.ActorMsgsCountStates, ActorMsgsCountState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsCountMap: actorMsgCountMap})
							mutex.Unlock()
						}
					}
				}
			} else if refreshList.storeType == "add" {
				var allActorsRes []model.AllActorsRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetAllActorsForBlockMessageAggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store add [latestStartEpoch,endEpoch)
						actorMsgCountMap := refreshList.previousState.(map[string]int64)

						if len(allActorsRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorMsgsCountStates[previousLength] = ActorMsgsCountState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsCountMap: actorMsgCountMap}
							mutex.Unlock()
						} else {
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
								actorMsgCountMap[actor] += count
							}

							mutex.Lock()
							ds.ActorMsgsCountStates[previousLength] = ActorMsgsCountState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorMsgsCountMap: actorMsgCountMap}
							mutex.Unlock()
						}
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	return nil
}

func RefreshActorTransferMsgs(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "ActorTransferMsgs")

	start := time.Now()

	sort.Slice(ds.ActorTransfersCountStates, func(i, j int) bool {
		return ds.ActorTransfersCountStates[i].StartEpoch < ds.ActorTransfersCountStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.ActorTransfersCountStates) - 1
	if len(ds.ActorTransfersCountStates) > 0 {
		latestStartEpoch = ds.ActorTransfersCountStates[previousLength].StartEpoch
		latestEndEpoch = ds.ActorTransfersCountStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh ActorTransfersCountMap successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.ActorTransfersCountStates[previousLength].ActorTransfersCountMap, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var allActorsRes []model.AllActorsRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfTransfersForActor2Aggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store new [startEpoch,endEpoch)
						actorTransfersCountMap := make(map[string]int64)
						if len(allActorsRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorTransfersCountStates = append(ds.ActorTransfersCountStates, ActorTransfersCountState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorTransfersCountMap: actorTransfersCountMap})
							mutex.Unlock()
						} else {
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
								actorTransfersCountMap[actor] += count
							}

							mutex.Lock()
							ds.ActorTransfersCountStates = append(ds.ActorTransfersCountStates, ActorTransfersCountState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, ActorTransfersCountMap: actorTransfersCountMap})
							mutex.Unlock()
						}
					}
				}

			} else if refreshList.storeType == "add" {
				var allActorsRes []model.AllActorsRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfTransfersForActor2Aggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "ExecTrace"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store add [latestStartEpoch,endEpoch)
						actorTransfersCountMap := refreshList.previousState.(map[string]int64)
						if len(allActorsRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.ActorTransfersCountStates[len(ds.ActorTransfersCountStates)-1] = ActorTransfersCountState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorTransfersCountMap: actorTransfersCountMap}
							mutex.Unlock()
						} else {
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
								actorTransfersCountMap[actor] += count
							}

							mutex.Lock()
							ds.ActorTransfersCountStates[previousLength] = ActorTransfersCountState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, ActorTransfersCountMap: actorTransfersCountMap}
							mutex.Unlock()
						}
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	return nil
}

func RefreshMinedMsgsMaps(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "MinedMsgsMap")

	start := time.Now()

	sort.Slice(ds.MinedMsgsStates, func(i, j int) bool {
		return ds.MinedMsgsStates[i].StartEpoch < ds.MinedMsgsStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.MinedMsgsStates) - 1
	if len(ds.MinedMsgsStates) > 0 {
		latestStartEpoch = ds.MinedMsgsStates[previousLength].StartEpoch
		latestEndEpoch = ds.MinedMsgsStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh MinedMsgsMap successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.MinedMsgsStates[previousLength].MinedMsgsMap, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var minedCountForMinersRes []model.MinedCountForMinersRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetMinedCountForMinersAggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "BlockHeader"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store new [startEpoch,endEpoch)
						minedMsgsCountMap := make(map[string]int64)

						if len(minedCountForMinersRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.MinedMsgsStates = append(ds.MinedMsgsStates, MinedMsgsState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, MinedMsgsMap: minedMsgsCountMap})
							mutex.Unlock()
						} else {
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
								minedMsgsCountMap[miner] += count
							}

							mutex.Lock()
							ds.MinedMsgsStates = append(ds.MinedMsgsStates, MinedMsgsState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, MinedMsgsMap: minedMsgsCountMap})
							mutex.Unlock()
						}
					}
				}
			} else if refreshList.storeType == "add" {
				var minedCountForMinersRes []model.MinedCountForMinersRes
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetMinedCountForMinersAggregator())) // todo: ExecTrace
				if err != nil {
					//log.Errorf("parse for all actors failed, lastHeightForAllActors: %v, finalHeight: %v, err: %v", lastHeightForAllActors, finalHeight, err)
					return err
				}

				tableName := "BlockHeader"
				for _, col := range cols.Cols {
					if col != nil && col.Name() == tableName {
						// agg [startEpoch,endEpoch)
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

						// store add [latestStartEpoch,endEpoch)
						newMinedMsgsMap := refreshList.previousState.(map[string]int64) // 注意nil情况
						if len(minedCountForMinersRes) == 0 {
							rlog.Warnf("no actor between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
							mutex.Lock()
							ds.MinedMsgsStates[previousLength] = MinedMsgsState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, MinedMsgsMap: newMinedMsgsMap}
							mutex.Unlock()
						} else {
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
								newMinedMsgsMap[miner] += count
							}

							mutex.Lock()
							ds.MinedMsgsStates[previousLength] = MinedMsgsState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, MinedMsgsMap: newMinedMsgsMap}
							mutex.Unlock()
						}
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
	}

	return nil
}

func RefreshTransfersForLargeAmount(ctx context.Context, ds *DataBaseState, cols Collections, limit, interval int) error {
	rlog := log.With("refresh", "TransfersForLargeAmount")

	start := time.Now()

	sort.Slice(ds.TransfersLargeAmountStates, func(i, j int) bool {
		return ds.TransfersLargeAmountStates[i].StartEpoch < ds.TransfersLargeAmountStates[j].StartEpoch
	})

	latestStartEpoch, latestEndEpoch := ds.StartEpoch, ds.StartEpoch
	previousLength := len(ds.TransfersLargeAmountStates) - 1
	if len(ds.TransfersLargeAmountStates) > 0 {
		latestStartEpoch = ds.TransfersLargeAmountStates[previousLength].StartEpoch
		latestEndEpoch = ds.TransfersLargeAmountStates[previousLength].EndEpoch
	}

	defer func() {
		rlog.Infof("refresh TransfersLargeAmountCount successfully between %v and %v, elapsed: %v", latestEndEpoch, ds.EndEpoch, time.Now().Sub(start).String())
	}()

	if ds.EndEpoch <= latestEndEpoch {
		return nil
	}

	refreshLists := getRefreshLists(latestStartEpoch, latestEndEpoch, ds.EndEpoch, ds.TransfersLargeAmountStates[previousLength].TransfersLargeAmountCount, interval)

	lim := limiter.New(limit)
	var (
		ewg   multierror.Group
		mutex sync.Mutex
	)
	for i := range refreshLists {
		i := i
		refreshList := refreshLists[i]
		ewg.Go(func() error {
			if !lim.Acquire(context.TODO()) {
				return nil
			}

			defer func() {
				lim.Release(context.TODO())
			}()

			if refreshList.storeType == "new" {
				var countOfLargeAmountTransfers []model.CountOfLargeAmountTransfers
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfLargeAmountTransfersAggregator())) // todo: ExecTrace
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

						count := int64(0)
						if len(countOfLargeAmountTransfers) == 0 {
							rlog.Warnf("no large amount transfers between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
						} else {
							count = countOfLargeAmountTransfers[0].Count
						}

						mutex.Lock()
						ds.TransfersLargeAmountStates = append(ds.TransfersLargeAmountStates, TransfersLargeAmountState{StartEpoch: refreshList.startEpoch, EndEpoch: refreshList.endEpoch, TransfersLargeAmountCount: count})
						mutex.Unlock()
					}
				}
			} else if refreshList.storeType == "add" {
				var countOfLargeAmountTransfers []model.CountOfLargeAmountTransfers
				pipe, err := util.Parse(model.Ctx{StartEpoch: int64(refreshList.startEpoch), EndEpoch: int64(refreshList.endEpoch)}, string(monitor.GetCountOfLargeAmountTransfersAggregator())) // todo: ExecTrace
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

						// store add [latestStartEpoch,endEpoch)
						count := refreshList.previousState.(int64)
						if len(countOfLargeAmountTransfers) == 0 {
							rlog.Warnf("no large amount transfers between %v and %v", refreshList.startEpoch, refreshList.endEpoch)
						} else {
							count += countOfLargeAmountTransfers[0].Count
						}

						mutex.Lock()
						ds.TransfersLargeAmountStates[previousLength] = TransfersLargeAmountState{StartEpoch: refreshList.latestStartEpoch, EndEpoch: refreshList.endEpoch, TransfersLargeAmountCount: count}
						mutex.Unlock()
					}
				}
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return err
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
