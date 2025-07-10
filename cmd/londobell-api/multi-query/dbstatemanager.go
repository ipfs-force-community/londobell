package multiquery

import (
	"context"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/hashicorp/go-multierror"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	config2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

const DefaultRPCListenAddr = "/ip4/127.0.0.1/tcp/12346"

var (
	DBStateManager DataBaseStateManager
	ActorIDMap     = make(map[string]string) // robust/delegated: ID
	ALock          sync.RWMutex              // todo: for ActorIDMap

	ErrNotFoundInDBCollectionsMap = fmt.Errorf("not found in DBCollectionsMap")
	ErrNotFound                   = fmt.Errorf("not found")
)

type DataBaseStateManager struct {
	fx.In
	Segment      *segment.Segment
	DBStateCache *DataBaseStateCache
	DBCfg        *config2.DBCollectionsConfigMgr
}

// todo: only for formal & colds
func (dbsm *DataBaseStateManager) GetState(ctx context.Context, dsn string) (*segment.State, bool, error) {
	state, ok := dbsm.DBStateCache.GetState(dsn)
	if !ok {
		return dbsm.RefreshState(ctx, dsn)
	}

	return state, true, nil
}

// 从数据库获取数据,刷新缓存
func (dbsm *DataBaseStateManager) RefreshState(ctx context.Context, dsn string) (*segment.State, bool, error) {
	state, found, err := dbsm.Segment.GetState(ctx, dsn)
	if err != nil {
		return nil, false, err
	}
	if !found {
		// todo
		return nil, false, nil
	}

	dbsm.DBStateCache.SetState(dsn, state)

	return state, true, nil

}

func (dbsm *DataBaseStateManager) GetDBState(ctx context.Context, dsn string) (*smodel.DBState, bool, error) {
	dbState, ok := dbsm.DBStateCache.GetDBState(dsn)
	if !ok {
		dbState, found, err := dbsm.Segment.GetDBState(ctx, dsn)
		if err != nil {
			return nil, false, err
		}
		if !found {
			// todo
			return nil, false, nil
		}

		dbsm.DBStateCache.FindAndUpdateDBState(dsn, dbState)

		return dbState, true, nil
	}

	return dbState, true, nil
}

//func (dbsm *DataBaseStateManager) GetDealState(ctx context.Context, dsn string) (smodel.DealState, bool, error) {
//	dealState, ok := dbsm.DBStateCache.GetDealState(dsn)
//	if !ok {
//		dealState, found, err := dbsm.Segment.GetDealState(ctx, dsn)
//		if err != nil {
//			return smodel.DealState{}, false, err
//		}
//
//		if !found {
//			return smodel.DealState{}, false, err
//		}
//
//		dbsm.DBStateCache.FindAndUpdateDealState(dsn, dealState)
//
//		return dealState, true, nil
//	}
//
//	return dealState, true, nil
//}

func (dbsm *DataBaseStateManager) GetBlockStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	blockStates, ok := dbsm.DBStateCache.GetBlockStates(dsn)
	if !ok {
		blockStates, err := dbsm.Segment.GetBlockStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		// todo: nil
		if err := dbsm.DBStateCache.SetBlockStates(dsn, blockStates); err != nil {
			return nil, err
		}

		return blockStates, nil
	}

	return blockStates, nil
}

func (dbsm *DataBaseStateManager) GetBlockMethodStates(ctx context.Context, dsn string, methodName string) ([]smodel.SegmentState, error) {
	blockMethodStates, ok := dbsm.DBStateCache.GetBlockMethodStates(dsn, methodName)
	if !ok {
		allBlockMethodStates, err := dbsm.Segment.GetAllBlockMethodStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetBlockMethodStates(dsn, allBlockMethodStates); err != nil {
			return nil, err
		}

		blockMethodStates, _ := dbsm.DBStateCache.GetBlockMethodStates(dsn, methodName)
		return blockMethodStates, nil
	}

	return blockMethodStates, nil
}

func (dbsm *DataBaseStateManager) GetAllBlockMethodStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	blockMethodStates, ok := dbsm.DBStateCache.GetAllBlockMethodStates(dsn)
	if !ok {
		blockMethodStates, err := dbsm.Segment.GetAllBlockMethodStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetBlockMethodStates(dsn, blockMethodStates); err != nil {
			return nil, err
		}

		blockMethodStates, _ = dbsm.DBStateCache.GetAllBlockMethodStates(dsn)
		return blockMethodStates, nil
	}

	return blockMethodStates, nil
}

func (dbsm *DataBaseStateManager) GetActorStates(ctx context.Context, dsn string, actorID string) ([]smodel.SegmentState, error) {
	actorStates, ok := dbsm.DBStateCache.GetActorStates(dsn, actorID)
	if !ok {
		allActorStates, err := dbsm.Segment.GetAllActorStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetActorStates(dsn, allActorStates); err != nil {
			return nil, err
		}

		actorStates, _ := dbsm.DBStateCache.GetActorStates(dsn, actorID)
		return actorStates, nil
	}

	return actorStates, nil
}

func (dbsm *DataBaseStateManager) GetAllActorStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	actorStates, ok := dbsm.DBStateCache.GetAllActorStates(dsn)
	if !ok {
		actorStates, err := dbsm.Segment.GetAllActorStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetActorStates(dsn, actorStates); err != nil {
			return nil, err
		}

		return actorStates, nil
	}

	return actorStates, nil
}

func (dbsm *DataBaseStateManager) GetActorMethodStates(ctx context.Context, dsn, actorID, methodName string) ([]smodel.SegmentState, error) {
	actorMethodStates, ok := dbsm.DBStateCache.GetActorMethodStates(dsn, actorID, methodName)
	if !ok {
		allActorMethodStates, err := dbsm.Segment.GetAllActorMethodStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		//if !found {
		//	return smodel.SegmentState{}, ErrNotFound
		//}

		if err := dbsm.DBStateCache.SetActorMethodStates(dsn, allActorMethodStates); err != nil {
			return nil, err
		}

		actorMethodStates, _ := dbsm.DBStateCache.GetActorMethodStates(dsn, actorID, methodName)
		return actorMethodStates, nil
	}

	return actorMethodStates, nil
}

func (dbsm *DataBaseStateManager) GetAllActorMethodStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	actorMethodStates, ok := dbsm.DBStateCache.GetAllActorMethodStates(dsn)
	if !ok {
		actorMethodStates, err := dbsm.Segment.GetAllActorMethodStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetActorMethodStates(dsn, actorMethodStates); err != nil {
			return nil, err
		}

		return actorMethodStates, nil
	}

	return actorMethodStates, nil
}

func (dbsm *DataBaseStateManager) GetActorTransferStates(ctx context.Context, dsn string, actorID string) ([]smodel.SegmentState, error) {
	actorTransferStates, ok := dbsm.DBStateCache.GetActorTransferStates(dsn, actorID)
	if !ok {
		allActorTransferStates, err := dbsm.Segment.GetAllActorTransferStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetActorTransferStates(dsn, allActorTransferStates); err != nil {
			return nil, err
		}

		actorTransferStates, _ := dbsm.DBStateCache.GetActorTransferStates(dsn, actorID)
		return actorTransferStates, nil
	}

	return actorTransferStates, nil
}

func (dbsm *DataBaseStateManager) GetAllActorTransferStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	actorTransferStates, ok := dbsm.DBStateCache.GetAllActorTransferStates(dsn)
	if !ok {
		actorTransferStates, err := dbsm.Segment.GetAllActorTransferStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetActorTransferStates(dsn, actorTransferStates); err != nil {
			return nil, err
		}

		return actorTransferStates, nil
	}

	return actorTransferStates, nil
}

func (dbsm *DataBaseStateManager) GetMinedStates(ctx context.Context, dsn string, actorID string) ([]smodel.SegmentState, error) {
	minedStates, ok := dbsm.DBStateCache.GetMinedStates(dsn, actorID)
	if !ok {
		allMinedStates, err := dbsm.Segment.GetAllMinedStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetMinedStates(dsn, allMinedStates); err != nil {
			return nil, err
		}

		minedStates, _ := dbsm.DBStateCache.GetMinedStates(dsn, actorID)
		return minedStates, nil
	}

	return minedStates, nil
}

func (dbsm *DataBaseStateManager) GetAllMinedStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	minedStates, ok := dbsm.DBStateCache.GetAllMinedStates(dsn)
	if !ok {
		minedStates, err := dbsm.Segment.GetAllMinedStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetMinedStates(dsn, minedStates); err != nil {
			return nil, err
		}

		return minedStates, nil
	}

	return minedStates, nil
}

func (dbsm *DataBaseStateManager) GetLargeAmountTransferStates(ctx context.Context, dsn string) ([]smodel.SegmentState, error) {
	largeAmountTransferStates, ok := dbsm.DBStateCache.GetLargeAmountTransferStates(dsn)
	if !ok {
		largeAmountTransferStates, err := dbsm.Segment.GetLargeAmountTransferStates(ctx, dsn)
		if err != nil {
			return nil, err
		}

		if err := dbsm.DBStateCache.SetLargeAmountTransferStates(dsn, largeAmountTransferStates); err != nil {
			return nil, err
		}

		return largeAmountTransferStates, nil
	}

	return largeAmountTransferStates, nil
}

//func (dbsm *DataBaseStateManager) GetAllDealActorStates(ctx context.Context, dsn string) ([]smodel.SegmentDealState, error) {
//	dealActorStates, ok := dbsm.DBStateCache.GetAllDealActorStates(dsn)
//	if !ok {
//		dealActorStates, err := dbsm.Segment.GetAllDealActorStates(ctx, dsn)
//		if err != nil {
//			return nil, err
//		}
//
//		// todo: nil
//		if err := dbsm.DBStateCache.SetDealActorStates(dsn, dealActorStates); err != nil {
//			return nil, err
//		}
//
//		return dealActorStates, nil
//	}
//
//	return dealActorStates, nil
//}

//func (dbsm *DataBaseStateManager) GetDealActorStates(ctx context.Context, dsn string, actorID string) ([]smodel.SegmentDealState, error) {
//	dealActorStates, ok := dbsm.DBStateCache.GetDealActorStates(dsn, actorID)
//	if !ok {
//		allDealActorStates, err := dbsm.Segment.GetAllDealActorStates(ctx, dsn)
//		if err != nil {
//			return nil, err
//		}
//
//		if err := dbsm.DBStateCache.SetDealActorStates(dsn, allDealActorStates); err != nil {
//			return nil, err
//		}
//
//		dealActorStates, _ := dbsm.DBStateCache.GetDealActorStates(dsn, actorID)
//		return dealActorStates, nil
//	}
//
//	return dealActorStates, nil
//}

func (dbsm *DataBaseStateManager) GetCfgLastModifyTime() int64 {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.LastModifyTime
}

func (dbsm *DataBaseStateManager) GetCfg() config2.Config {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg
}

func (dbsm *DataBaseStateManager) GetTmpCfg() config2.DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Tmp
}

func (dbsm *DataBaseStateManager) GetFormalCfg() config2.DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Formal
}

func (dbsm *DataBaseStateManager) GetColdsCfg() []config2.DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Colds
}

func (dbsm *DataBaseStateManager) UpdateColdsCfg(db config2.DB) bool {
	var exist bool
	colds := dbsm.GetColdsCfg()

	for _, cold := range colds {
		if db.Equals(cold) {
			exist = true
		}
	}

	if !exist {
		dbsm.DBCfg.DBCollectionsConfigLk.Lock()
		defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

		dbsm.DBCfg.Cfg.Colds = append(dbsm.DBCfg.Cfg.Colds, db)
		return exist
	}

	return exist
}

func (dbsm *DataBaseStateManager) ReplaceColdsCfg(dbs []config2.DB) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.Cfg.Colds = dbs

	return
}

func (dbsm *DataBaseStateManager) GetDBCollections(url string) (config2.Collections, bool) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	cols, ok := dbsm.DBCfg.DBCollectionsMap[url]
	log.Infof("GetDBCollections: url: %s, ok: %v, len: %d", url, ok, len(cols.Cols))
	if ok {
		return cols, true
	}

	return config2.Collections{}, false
}

func (dbsm *DataBaseStateManager) SetConfig(cfg config2.Config) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.Cfg = cfg
}

func (dbsm *DataBaseStateManager) UpdateDBCollectionsMap(url string, collections config2.Collections) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.DBCollectionsMap[url] = collections
}

type Boundrary struct {
	Start abi.ChainEpoch `bson:"start" json:"Start"`
	End   abi.ChainEpoch `bson:"end" json:"End"`
}

type DealRange struct {
	Start uint64
	End   uint64
}

func FirstLoad(ctx context.Context, dbsm *DataBaseStateManager) error {
	if err := dbsm.LoadDBCollectionsMap(ctx); err != nil {
		return err
	}

	if err := dbsm.LoadDBStateCache(ctx); err != nil {
		return err
	}

	return nil
}

func Reload(ctx context.Context, dbsm *DataBaseStateManager, cfgPath string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := dbsm.MonitorConfig(ctx, cfgPath); err != nil {
				log.Error(err)
				continue
			}
		}
	}
}

// config改变时才reload
func (dbsm *DataBaseStateManager) MonitorConfig(ctx context.Context, cfgPath string) error {
	file, err := os.Open(cfgPath)
	if err != nil {
		log.Errorf("db config path %v error: %v", cfgPath, err)
		return err
	}

	defer file.Close()

	lastModifyTime := dbsm.GetCfgLastModifyTime()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	curModifyTime := fileInfo.ModTime().Unix()
	if curModifyTime > lastModifyTime {
		log.Infof("curModifyTime %v > lastModifyTime %v", curModifyTime, lastModifyTime)
		// reload config
		cfg := config2.Config{}
		_, err = config.FromReader(file, &cfg)
		if err != nil {
			return err
		}

		cfg.LastModifyTime = time.Now().Unix()
		err = config2.WriteToConfig(cfgPath, cfg)
		if err != nil {
			return err
		}

		dbsm.SetConfig(cfg)

		if err := dbsm.LoadDBCollectionsMap(ctx); err != nil {
			return err
		}

		if err := dbsm.LoadDBStateCache(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (dbsm *DataBaseStateManager) LoadDBCollectionsMap(ctx context.Context) error {
	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()
	tmp := dbsm.GetTmpCfg()

	for _, db := range append(colds, formal) {
		if db.IsInvalidDB() {
			log.Warnf("db %v is invalid", db)
			continue
		}
		log.Infof("LoadDBCollectionsMap: %v", db)

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(db.Url()).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
		if err != nil {
			return err
		}
		//defer client.Disconnect(ctx) //nolint:errcheck // todo: config更新后连接过多？

		database := client.Database(db.Name())
		traceCol := database.Collection("ExecTrace")
		actorBalanceCol := database.Collection("ActorBalance")
		finalHeightCol := database.Collection("FinalHeight")
		minerSectorHealthCol := database.Collection("MinerSectorHealth")
		tipSetCol := database.Collection("Tipset")
		actorStateCol := database.Collection("ActorState")
		minerFundsCol := database.Collection("MinerFunds")
		claimedPowerCol := database.Collection("ClaimedPower")
		newDealProposalCol := database.Collection("NewDealProposal")
		messageCol := database.Collection("Message")
		//messageBlockCol := database.Collection("MessageBlock")
		blockMessageCol := database.Collection("BlockMessage")
		blockHeaderCol := database.Collection("BlockHeader")
		orphanBlockCol := database.Collection("OrphanBlock")
		actorMessageCol := database.Collection("ActorMessage")
		ethHashCol := database.Collection("EthHash")
		eventsRootCol := database.Collection("EventsRoot")
		stateFinalHeightCol := database.Collection("StateFinalHeight")
		evmInitCodeCol := database.Collection("EvmInitCode")
		actorEventCodeCol := database.Collection("ActorEvent")
		actorAddressCol := database.Collection("ActorAddress")
		createMessageCol := database.Collection("CreateMessage")
		changedSectorCol := database.Collection("ChangedSector")
		changedActorCol := database.Collection("ChangedActor")
		changedClaimCol := database.Collection("ChangedClaim")
		changedDealStateCol := database.Collection("ChangedDealState")
		filSupplyCol := database.Collection("FilSupply")
		cols := make([]*mongo.Collection, 0)
		cols = append(cols, traceCol, actorBalanceCol, finalHeightCol, minerSectorHealthCol, tipSetCol, actorStateCol, minerFundsCol, claimedPowerCol, newDealProposalCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, evmInitCodeCol, actorEventCodeCol, actorAddressCol, createMessageCol,
			changedSectorCol, changedActorCol, filSupplyCol, changedClaimCol, changedDealStateCol, orphanBlockCol)
		dbsm.UpdateDBCollectionsMap(db.Url(), config2.Collections{DB: database, Cols: cols})
	}

	if tmp.IsInvalidDB() {
		log.Warnf("db %v is invalid", tmp)
		return nil
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(tmp.Url()).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
	if err != nil {
		return err
	}
	//defer client.Disconnect(ctx) //nolint:errcheck

	database := client.Database(tmp.Name())
	traceCol := database.Collection("ExecTrace")
	tipSetCol := database.Collection("Tipset")
	messageCol := database.Collection("Message")
	//messageBlockCol := database.Collection("MessageBlock")
	blockMessageCol := database.Collection("BlockMessage")
	blockHeaderCol := database.Collection("BlockHeader")
	orphanBlockCol := database.Collection("OrphanBlock")
	actorMessageCol := database.Collection("ActorMessage")
	createMessageCol := database.Collection("CreateMessage")
	ethHashCol := database.Collection("EthHash")
	eventsRootCol := database.Collection("EventsRoot")
	stateFinalHeightCol := database.Collection("StateFinalHeight")
	evmInitCodeCol := database.Collection("EvmInitCode")
	actorEventCodeCol := database.Collection("ActorEvent")

	cols := make([]*mongo.Collection, 0)
	cols = append(cols, traceCol, tipSetCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, evmInitCodeCol, actorEventCodeCol, createMessageCol, orphanBlockCol)
	dbsm.UpdateDBCollectionsMap(tmp.Url(), config2.Collections{DB: database, Cols: cols})

	return nil
}

func (dbsm *DataBaseStateManager) LoadDBStateCache(ctx context.Context) error {
	colds := dbsm.GetColdsCfg()
	formal := dbsm.GetFormalCfg()

	// ewg
	for i := range colds {
		i := i
		cold := colds[i]
		if cold.IsInvalidDB() {
			log.Warnf("db %v is invalid", cold)
			continue
		}

		_, found, err := dbsm.GetState(ctx, cold.Url())
		if err != nil {
			return err
		}

		if !found {
			return fmt.Errorf("state of dsn %v not found, please run cfgUpdateCmd firstly", cold.Url())
		}
	}

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
	} else {
		_, found, err := dbsm.GetState(ctx, formal.Url())
		if err != nil {
			return err
		}

		if !found {
			return fmt.Errorf("state of dsn %v not found, please run cfgUpdateCmd firstly", formal.Url())
		}
	}

	return nil
}

func GetCollectionsForDB(ctx context.Context, db config2.DB) (config2.Collections, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(db.Url()).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
	if err != nil {
		return config2.Collections{}, err
	}
	//defer client.Disconnect(ctx) //nolint:errcheck
	database := client.Database(db.Name())

	// todo: tmp有些库没有，先new应该不要紧，不用区分tmp?
	traceCol := database.Collection("ExecTrace")
	actorBalanceCol := database.Collection("ActorBalance")
	finalHeightCol := database.Collection("FinalHeight")
	minerSectorHealthCol := database.Collection("MinerSectorHealth")
	tipSetCol := database.Collection("Tipset")
	actorStateCol := database.Collection("ActorState")
	minerFundsCol := database.Collection("MinerFunds")
	claimedPowerCol := database.Collection("ClaimedPower")
	newDealProposalCol := database.Collection("NewDealProposal")
	messageCol := database.Collection("Message")
	//messageBlockCol := database.Collection("MessageBlock")
	blockMessageCol := database.Collection("BlockMessage")
	blockHeaderCol := database.Collection("BlockHeader")
	orphanBlockCol := database.Collection("OrphanBlock")
	actorMessageCol := database.Collection("ActorMessage")
	ethHashCol := database.Collection("EthHash")
	eventsRootCol := database.Collection("EventsRoot")
	stateFinalHeightCol := database.Collection("StateFinalHeight")
	evmInitCodeCol := database.Collection("EvmInitCode")
	actorEventCodeCol := database.Collection("ActorEvent")
	actorAddressCol := database.Collection("ActorAddress")
	createMessageCol := database.Collection("CreateMessage")
	changedSectorCol := database.Collection("ChangedSector")
	changedActorCol := database.Collection("ChangedActor")
	changedClaimCol := database.Collection("ChangedClaim")
	changedDealStateCol := database.Collection("ChangedDealState")
	filSupplyCol := database.Collection("FilSupply")
	cols := make([]*mongo.Collection, 0)
	cols = append(cols, traceCol, actorBalanceCol, finalHeightCol, minerSectorHealthCol, tipSetCol, actorStateCol, minerFundsCol, claimedPowerCol, newDealProposalCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, evmInitCodeCol, actorEventCodeCol, actorAddressCol, createMessageCol,
		changedSectorCol, changedActorCol, filSupplyCol, changedClaimCol, newDealProposalCol, changedDealStateCol, orphanBlockCol)

	return config2.Collections{DB: database, Cols: cols}, nil
}

func GetTipSetStartEpoch(ctx context.Context, cols config2.Collections) (abi.ChainEpoch, error) {
	var boundaryRes []Boundrary
	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetBoundaryOfDBAggregator()))
	if err != nil {
		return 0, err
	}

	tableName := "Tipset"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return 0, err
			}

			err = cur.All(ctx, &boundaryRes)
			if err != nil {
				return 0, err
			}

			if len(boundaryRes) == 0 {
				return 0, fmt.Errorf("no data for Tipset")
			}

			boundary := boundaryRes[0]

			return boundary.Start, nil
		}
	}

	return 0, fmt.Errorf("no TipSet table")

}

func GetEndEpoch(ctx context.Context, cols config2.Collections) (abi.ChainEpoch, error) {
	var finalHeightRes []model.FinalHeightRes
	pipe, err := util.Parse(model.Ctx{}, string(monitor.GetFinalHeightAggregator()))
	if err != nil {
		return 0, err
	}

	tableName := "FinalHeight"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return 0, err
			}
			err = cur.All(ctx, &finalHeightRes)
			if err != nil {
				return 0, err
			}

			if len(finalHeightRes) == 0 {
				return 0, fmt.Errorf("no data in FinalHeight")
			}

			endEpoch := finalHeightRes[0].Epoch + 1

			return endEpoch, nil
		}
	}

	return 0, fmt.Errorf("no FinalHeight table")
}

// tmp不管
func (dbsm *DataBaseStateManager) GetBoundaryForDB(ctx context.Context, cols config2.Collections, dbType smodel.DType) (Boundrary, error) {
	cfg := dbsm.GetCfg()
	countUtils := make([]CountUtil, 0)
	for _, cold := range cfg.Colds {
		if cold.IsInvalidDB() {
			continue
		}

		state, found, err := dbsm.GetState(ctx, cold.Url())
		if err != nil {
			return Boundrary{}, fmt.Errorf("load dbState for cold %v failed", cold.Url())
		}

		if !found {
			return Boundrary{}, fmt.Errorf("state of dsn %v not found, please run cfgUpdateCmd firstly", cold.Url())
		}

		countUtils = append(countUtils, CountUtil{Start: int64(state.GetStartEpoch()), End: int64(state.GetEndEpoch())})
	}

	// 逆序排序
	sort.Slice(countUtils, func(i, j int) bool {
		return countUtils[i].End > countUtils[j].End
	})

	// tipset最早高度，此高度所有表数据一定是全的
	startEpoch, err := GetTipSetStartEpoch(ctx, cols)
	if err != nil {
		return Boundrary{}, err
	}

	endEpoch, err := GetEndEpoch(ctx, cols)
	if err != nil {
		return Boundrary{}, err
	}

	switch dbType {
	case smodel.Formal:
		// end: finalheight+1

		// start
		if len(countUtils) != 0 {
			latestEndEpoch := countUtils[0].End
			// 判断formal是否包含latestEndEpoch, 即formal.start <= latestEndEpoch
			// 包含则start: latestEndEpoch, 不包含则tipset
			if startEpoch <= abi.ChainEpoch(latestEndEpoch) {
				startEpoch = abi.ChainEpoch(latestEndEpoch)
			}
		}

		return Boundrary{
			Start: startEpoch,
			End:   endEpoch,
		}, nil

		// [start, end) [1,6) [3,6) [3,4) [1,4)  [2,4)（大，小】 [11,2)
		// [2,3)  [5,8)  [10,11)  [13, ...)   [2,5) [8,10) [11,13)
	case smodel.Cold:
		//// 添加上下边界
		//minStartEpoch := int64(0)
		//if len(countUtils) > 0 {
		//	minStartEpoch = countUtils[len(countUtils)-1].Start
		//}

		if !cfg.Formal.IsInvalidDB() {
			formalState, found, err := dbsm.GetState(ctx, cfg.Formal.Url())
			if err != nil {
				return Boundrary{}, fmt.Errorf("load dbState for formal %v failed", cfg.Formal.Url())
			}

			if !found {
				return Boundrary{}, fmt.Errorf("state of dsn %v not found", cfg.Formal.Url())
			}

			countUtils = append(countUtils, CountUtil{Start: int64(formalState.GetStartEpoch()), End: math.MaxInt64})
			// 逆序排序
			sort.Slice(countUtils, func(i, j int) bool {
				return countUtils[i].End > countUtils[j].End
			})
		}

		countUtils = append(countUtils, CountUtil{Start: 0, End: 0}, CountUtil{Start: math.MaxInt64, End: math.MaxInt64})
		sort.Slice(countUtils, func(i, j int) bool {
			return countUtils[i].Start > countUtils[j].Start
		})

		// 找出不连续的段
		discontinuousSegment := make([]CountUtil, 0)
		for i := 1; i < len(countUtils); i++ {
			if countUtils[i-1].Start != countUtils[i].End {
				discontinuousSegment = append(discontinuousSegment, CountUtil{
					Start: countUtils[i].End,
					End:   countUtils[i-1].Start,
				})
			}
		}

		sort.Slice(discontinuousSegment, func(i, j int) bool {
			return discontinuousSegment[i].End > discontinuousSegment[j].End
		})

		for _, seg := range discontinuousSegment {
			start := abi.ChainEpoch(math.Max(float64(startEpoch), float64(seg.Start)))
			end := abi.ChainEpoch(math.Min(float64(endEpoch), float64(seg.End)))
			if start >= end {
				continue
			}

			return Boundrary{Start: start, End: end}, nil
		}

		return Boundrary{}, fmt.Errorf("no needed boundary")

		//// 顺着加
		//if !reverse {
		//	// start
		//	if len(countUtils) != 0 {
		//		latestEndEpoch := countUtils[len(countUtils)-1].End
		//		// 判断cold是否包含latestEndEpoch, 即cold.start <= latestEndEpoch
		//		// 包含则start: latestEndEpoch, 不包含则报错（不能断开）
		//		if start <= abi.ChainEpoch(latestEndEpoch) {
		//			start = abi.ChainEpoch(latestEndEpoch)
		//		} else {
		//			return Boundrary{}, fmt.Errorf("add discontinuous cold in sequential order, start: %v, latestEndEpoch: %v", start, latestEndEpoch)
		//		}
		//	}
		//
		//	// end
		//	if IsInvalidDB(cfg.Formal) {
		//		formalState, found, err := dbsm.Stm.LoadDataBaseState(cfg.Formal.Url)
		//		if err != nil || !found {
		//			return Boundrary{}, fmt.Errorf("load dbState for formal %v failed", cfg.Formal.Url)
		//		}
		//
		//		formalStartEpoch := formalState.StartEpoch
		//		// 判断cold是否抵达formalStartEpoch，即cold.finalheight+1 >= formalStartEpoch
		//		// 抵达则end: formalStartEpoch, 否则finalheight+1
		//		if end >= formalStartEpoch {
		//			end = formalStartEpoch
		//		}
		//	}
		//
		//	return Boundrary{
		//		Start: start,
		//		End:   end,
		//	}, nil
		//}
		//
		//// 逆着加
		//// start:tipset
		//
		//// end
		//if len(countUtils) == 0 {
		//	if IsInvalidDB(cfg.Formal) {
		//		formalState, found, err := DBStateManager.Stm.LoadDataBaseState(cfg.Formal.Url)
		//		if err != nil || !found {
		//			return Boundrary{}, fmt.Errorf("load dbState for formal %v failed", cfg.Formal.Url)
		//		}
		//
		//		formalStartEpoch := formalState.StartEpoch
		//		// 判断cold是否抵达formalStartEpoch，即cold.finalheight+1 >= formalStartEpoch
		//		// 抵达则end: formalStartEpoch, 否则finalheight+1
		//		if end >= formalStartEpoch {
		//			end = formalStartEpoch
		//		}
		//	}
		//} else {
		//	latestStartEpoch := countUtils[0].Start
		//	// 判断cold是否抵达latestStartEpoch, 即cold.finalheight+1 >=latestStartEpoch
		//	// 抵达则end: latestStartEpoch, 否则报错
		//	if end >= abi.ChainEpoch(latestStartEpoch) {
		//		end = abi.ChainEpoch(latestStartEpoch)
		//	} else {
		//		return Boundrary{}, fmt.Errorf("add discontinuous cold in reverse order, end: %v, latestStartEpoch: %v", end, latestStartEpoch)
		//	}
		//}
		//
		//return Boundrary{Start: start, End: end}, nil

	default:
		return Boundrary{}, fmt.Errorf("invalid db type: %v", dbType)
	}
}

func (dbsm *DataBaseStateManager) FirstSetDataBaseState(ctx context.Context, newDB config2.DB, dbType smodel.DType, interval int64) error {
	flog := log.With("FirstSetDataBaseState", newDB)

	cols, err := GetCollectionsForDB(ctx, newDB)
	if err != nil {
		log.Errorf("get collections for DB %v failed: %v", newDB, cols)
		return err
	}

	boundary, err := dbsm.GetBoundaryForDB(ctx, cols, dbType)
	if err != nil {
		log.Errorf("get boundary for db %v failed: %v", newDB, err)
		return err
	}

	flog.Infow("get boundary for db", "db", newDB, "boundary", boundary)

	state := segment.DefaultState(newDB.Url(), dbType, interval, boundary.Start, boundary.End, 0, 0)
	if err != nil {
		return err
	}

	addUpState := segment.NewAddUpState(*state)

	if dbType == smodel.Formal || dbType == smodel.Cold {
		nextEndEpoch := int64(state.GetEndEpoch())

		var ewg multierror.Group
		for i := range addupes {
			i := i
			addup := addupes[i]
			ewg.Go(func() error {
				if err := addup(ctx, flog, state, addUpState, cols, dbsm.Segment, nextEndEpoch); err != nil {
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
		if err := AddUpDBState(ctx, flog, state, addUpState, cols, dbsm.Segment, nextEndEpoch); err != nil {
			log.Errorf("AddUpDBState faild for nextEndEpoch: %v", nextEndEpoch)
			return err
		}
	}

	return nil
}

// todo: update addupstate
func (dbsm *DataBaseStateManager) UpdateAllState(ctx context.Context, nextEndEpoch int64, state *segment.State, addUpState *segment.AddUpState, cols common.Collections) error {
	dlog := log.With("UpdateAllState", nextEndEpoch)

	var ewg multierror.Group
	for i := range addupes {
		i := i
		addup := addupes[i]
		ewg.Go(func() error {
			if err := addup(ctx, dlog, state, addUpState, cols, dbsm.Segment, nextEndEpoch); err != nil {
				return err
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		log.Errorf("RefreshFormalDataBaseState failed: %v", err)
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) UpdateBlockState(ctx context.Context, nextEndEpoch int64, state *segment.State, addUpState *segment.AddUpState, cols common.Collections) error {
	dlog := log.With("UpdateBlockState", nextEndEpoch)

	err := dbsm.Segment.AddUpBlockState(ctx, dlog, nextEndEpoch, state, addUpState, cols)
	if err != nil {
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) UpdateBlockMethodState(ctx context.Context, nextEndEpoch int64, state *segment.State, addUpState *segment.AddUpState, cols common.Collections) error {
	dlog := log.With("UpdateBlockMethodState", nextEndEpoch)

	err := dbsm.Segment.AddUpBlockMethodStates(ctx, dlog, nextEndEpoch, state, addUpState, cols)
	if err != nil {
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) DeleteAllState(ctx context.Context, db config2.DB) error {
	dlog := log.With("DeleteAllState", db)

	err := dbsm.Segment.DeleteDBState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	err = dbsm.Segment.DeleteBlockState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	err = dbsm.Segment.DeleteBlockMethodState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) DeleteDBState(ctx context.Context, db config2.DB) error {
	dlog := log.With("DeleteDBState", db)

	err := dbsm.Segment.DeleteDBState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) DeleteBlockState(ctx context.Context, db config2.DB) error {
	dlog := log.With("DeleteBlockState", db)

	err := dbsm.Segment.DeleteBlockState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	return nil
}

func (dbsm *DataBaseStateManager) DeleteBlockMethodState(ctx context.Context, db config2.DB) error {
	dlog := log.With("DeleteBlockMethodState", db)

	err := dbsm.Segment.DeleteBlockMethodState(ctx, dlog, db.Url())
	if err != nil {
		return err
	}

	return nil
}
