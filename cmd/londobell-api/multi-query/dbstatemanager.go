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

	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node/config"

	"github.com/filecoin-project/go-state-types/abi"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const DefaultRPCListenAddr = "/ip4/127.0.0.1/tcp/12346"

var (
	DBStateManager DataBaseStateManager
	ActorIDMap     = make(map[string]string) // robust/delegated: ID
	ALock          sync.RWMutex              // todo: for ActorIDMap

	ErrNotFoundInDBCollectionsMap = fmt.Errorf("not found in DBCollectionsMap")
)

type DataBaseStateManager struct {
	fx.In
	Stm          *StateManager
	DBStateCache *DataBaseStateCache
	DBCfg        *DBCollectionsConfigMgr

	RPath RepoPath
}

type DataBaseStateCache struct {
	cache map[string]*DataBaseState
	clk   sync.RWMutex
}

func NewDataBaseStateCache() *DataBaseStateCache {
	return &DataBaseStateCache{
		cache: make(map[string]*DataBaseState),
	}
}

func (dbsc *DataBaseStateCache) GetDataBase(url string) (*DataBaseState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if dbState, ok := dbsc.cache[url]; ok {
		return dbState, true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) SetDataBase(url string, dbState *DataBaseState) {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	dbsc.cache[url] = dbState
}

func (dbsm *DataBaseStateManager) GetRepoPath() RepoPath {
	return dbsm.RPath
}

func (dbsm *DataBaseStateManager) GetDataBase(url string) (*DataBaseState, error) {
	var dbState *DataBaseState
	if dsc, ok := dbsm.DBStateCache.GetDataBase(url); !ok {
		ds, found, err := dbsm.Stm.LoadDataBaseState(url)
		if err != nil {
			return nil, err
		}
		if !found {
			// todo
			return nil, fmt.Errorf("db %v not found", url)
		}

		dbsm.DBStateCache.SetDataBase(url, &ds)
		dbState = &ds
	} else {
		dbState = dsc
	}

	return dbState, nil
}

func (dbsm *DataBaseStateManager) GetCfgLastModifyTime() int64 {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.LastModifyTime
}

func (dbsm *DataBaseStateManager) GetCfg() Config {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg
}

func (dbsm *DataBaseStateManager) GetTmpCfg() DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Tmp
}

func (dbsm *DataBaseStateManager) GetFormalCfg() DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Formal
}

func (dbsm *DataBaseStateManager) GetColdsCfg() []DB {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	return dbsm.DBCfg.Cfg.Colds
}

func (dbsm *DataBaseStateManager) UpdateColdsCfg(db DB) bool {
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

func (dbsm *DataBaseStateManager) ReplaceColdsCfg(dbs []DB) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.Cfg.Colds = dbs

	return
}

func (dbsm *DataBaseStateManager) GetDBCollections(url string) (Collections, bool) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	cols, ok := dbsm.DBCfg.DBCollectionsMap[url]
	if ok {
		return cols, true
	}

	return Collections{}, false
}

func (dbsm *DataBaseStateManager) SetConfig(cfg Config) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.Cfg = cfg
}

func (dbsm *DataBaseStateManager) UpdateDBCollectionsMap(url string, collections Collections) {
	dbsm.DBCfg.DBCollectionsConfigLk.Lock()
	defer dbsm.DBCfg.DBCollectionsConfigLk.Unlock()

	dbsm.DBCfg.DBCollectionsMap[url] = collections
}

type Boundrary struct {
	Start abi.ChainEpoch `bson:"start" json:"Start"`
	End   abi.ChainEpoch `bson:"end" json:"End"`
}

func FirstLoad(ctx context.Context, dbsm *DataBaseStateManager) error {
	if err := dbsm.LoadDBCollectionsMap(ctx); err != nil {
		return err
	}

	if err := dbsm.LoadDBStateCache(); err != nil {
		return err
	}

	return nil
}

func Reload(ctx context.Context, dbsm *DataBaseStateManager) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := dbsm.MonitorConfig(ctx, ConfigFilePath(dbsm.GetRepoPath())); err != nil {
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
		cfg := Config{}
		_, err = config.FromReader(file, &cfg)
		if err != nil {
			return err
		}

		cfg.LastModifyTime = time.Now().Unix()
		err = WriteToConfig(cfgPath, cfg)
		if err != nil {
			return err
		}

		dbsm.SetConfig(cfg)

		if err := dbsm.LoadDBCollectionsMap(ctx); err != nil {
			return err
		}

		if err := dbsm.LoadDBStateCache(); err != nil {
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
		dealProposalCol := database.Collection("DealProposal")
		messageCol := database.Collection("Message")
		//messageBlockCol := database.Collection("MessageBlock")
		blockMessageCol := database.Collection("BlockMessage")
		blockHeaderCol := database.Collection("BlockHeader")
		actorMessageCol := database.Collection("ActorMessage")
		ethHashCol := database.Collection("EthHash")
		eventsRootCol := database.Collection("EventsRoot")
		stateFinalHeightCol := database.Collection("StateFinalHeight")
		explicitMessageCol := database.Collection("ExplicitMessage")

		cols := make([]*mongo.Collection, 0)
		cols = append(cols, traceCol, actorBalanceCol, finalHeightCol, minerSectorHealthCol, tipSetCol, actorStateCol, minerFundsCol, claimedPowerCol, dealProposalCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, explicitMessageCol)
		dbsm.UpdateDBCollectionsMap(db.Url(), Collections{DB: database, Cols: cols})
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
	actorMessageCol := database.Collection("ActorMessage")
	ethHashCol := database.Collection("EthHash")
	eventsRootCol := database.Collection("EventsRoot")
	stateFinalHeightCol := database.Collection("StateFinalHeight")
	explicitMessageCol := database.Collection("ExplicitMessage")

	cols := make([]*mongo.Collection, 0)
	cols = append(cols, traceCol, tipSetCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, explicitMessageCol)
	dbsm.UpdateDBCollectionsMap(tmp.Url(), Collections{DB: database, Cols: cols})

	return nil
}

func (dbsm *DataBaseStateManager) LoadDBStateCache() error {
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

		if _, ok := dbsm.DBStateCache.GetDataBase(cold.Url()); !ok {
			dbState, found, err := dbsm.Stm.LoadDataBaseState(cold.Url())
			if err != nil {
				return err
			}

			if !found {
				return fmt.Errorf("db %v not found in DataBaseState, please run cfgUpdateCmd firstly", cold.Url())
			}

			dbsm.DBStateCache.SetDataBase(cold.Url(), &dbState)
		}
	}

	if formal.IsInvalidDB() {
		log.Warnf("db %v is invalid", formal)
	} else {
		if _, ok := dbsm.DBStateCache.GetDataBase(formal.Url()); !ok {
			_, found, err := dbsm.Stm.LoadDataBaseState(formal.Url())
			if err != nil {
				return err
			}

			// todo: 逐步递增加载formal
			if !found {
				return fmt.Errorf("db %v not found in DataBaseState, please run cfgUpdateCmd firstly", formal.Url())
			}
		}
	}

	return nil
}

func GetCollectionsForDB(ctx context.Context, db DB) (Collections, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(db.Url()).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
	if err != nil {
		return Collections{}, err
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
	dealProposalCol := database.Collection("DealProposal")
	messageCol := database.Collection("Message")
	//messageBlockCol := database.Collection("MessageBlock")
	blockMessageCol := database.Collection("BlockMessage")
	blockHeaderCol := database.Collection("BlockHeader")
	actorMessageCol := database.Collection("ActorMessage")
	ethHashCol := database.Collection("EthHash")
	eventsRootCol := database.Collection("EventsRoot")
	stateFinalHeightCol := database.Collection("StateFinalHeight")
	explicitMessageCol := database.Collection("ExplicitMessage")

	cols := make([]*mongo.Collection, 0)
	cols = append(cols, traceCol, actorBalanceCol, finalHeightCol, minerSectorHealthCol, tipSetCol, actorStateCol, minerFundsCol, claimedPowerCol, dealProposalCol, messageCol, blockMessageCol, blockHeaderCol, actorMessageCol, ethHashCol, eventsRootCol, stateFinalHeightCol, explicitMessageCol)

	return Collections{DB: database, Cols: cols}, nil
}

func GetTipSetStartEpoch(ctx context.Context, cols Collections) (abi.ChainEpoch, error) {
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

func GetEndEpoch(ctx context.Context, cols Collections) (abi.ChainEpoch, error) {
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
func (dbsm *DataBaseStateManager) GetBoundaryForDB(ctx context.Context, cols Collections, dbType string) (Boundrary, error) {
	cfg := dbsm.GetCfg()
	countUtils := make([]CountUtil, 0)
	for _, cold := range cfg.Colds {
		if cold.IsInvalidDB() {
			continue
		}

		dbState, ok, err := dbsm.Stm.LoadDataBaseState(cold.Url())
		if err != nil || !ok {
			return Boundrary{}, fmt.Errorf("load dbState for cold %v failed", cold.Url())
		}

		countUtils = append(countUtils, CountUtil{Start: int64(dbState.StartEpoch), End: int64(dbState.EndEpoch)})
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
	case "formal":
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
	case "cold":
		// 添加上下边界
		minStartEpoch := int64(0)
		if len(countUtils) > 0 {
			minStartEpoch = countUtils[len(countUtils)-1].Start
		}

		if !cfg.Formal.IsInvalidDB() {
			formalState, found, err := dbsm.Stm.LoadDataBaseState(cfg.Formal.Url())
			if err != nil || !found {
				return Boundrary{}, fmt.Errorf("load dbState for formal %v failed", cfg.Formal.Url())
			}

			countUtils = append(countUtils, CountUtil{Start: int64(formalState.StartEpoch), End: math.MaxInt64})
			// 逆序排序
			sort.Slice(countUtils, func(i, j int) bool {
				return countUtils[i].End > countUtils[j].End
			})
		}

		countUtils = append(countUtils, CountUtil{Start: 0, End: minStartEpoch}, CountUtil{Start: math.MaxInt64, End: math.MaxInt64})
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

func (dbsm *DataBaseStateManager) FirstSetDataBaseState(ctx context.Context, newDB DB, dbType string, formal, tmp bool) error {
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

	log.Infow("get boundary for db", "db", newDB, "boundary", boundary)

	dbState := DefaultDataBaseState(formal, tmp, boundary.Start, boundary.End)
	if err != nil {
		return err
	}

	if !tmp && !formal {
		log.Infow("begin RefreshDataBaseState...")
		//if err := RefreshDataBaseState(ctx, dbState, cols); err != nil {
		//	log.Errorf("refresh DataBaseState failed: %v", err)
		//	return err
		//}

		log.Infow("begin RefreshBlockMsgs...")
		count, err := RefreshBlockMsgs(ctx, dbState, cols, "", "")
		if err != nil {
			return err
		}

		dbState.BlockMsgsCount = count
		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshBlockMsgsByMethodName...")
		allBlockMethodNames, err := GetAllBlockMethodNamesMap(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
		if err != nil {
			return err
		}

		for _, allBlockMethodName := range allBlockMethodNames {
			dbState.BlockMsgsByMethodNameMap[allBlockMethodName.MethodName] += allBlockMethodName.Count
		}

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshActorMsgsByMethodName...")
		allActorsMethods, err := GetAllActorsMethods(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
		if err != nil {
			return err
		}

		for _, allActorMethods := range allActorsMethods {
			if _, ok := dbState.ActorMsgsByMethodNameMap[allActorMethods.ActorIDMethod.MethodName]; !ok {
				dbState.ActorMsgsByMethodNameMap[allActorMethods.ActorIDMethod.MethodName] = make(map[string]int64)
				dbState.ActorMsgsByMethodNameMap[allActorMethods.ActorIDMethod.MethodName][allActorMethods.ActorIDMethod.ActorID] = allActorMethods.Count
			} else {
				dbState.ActorMsgsByMethodNameMap[allActorMethods.ActorIDMethod.MethodName][allActorMethods.ActorIDMethod.ActorID] += allActorMethods.Count
			}
		}

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshActorMsgs...")
		allActorsMsgsCount, err := GetAllActorsMsgsCount(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
		if err != nil {
			return err
		}

		for _, allActorMsgsCount := range allActorsMsgsCount {
			dbState.ActorMsgsCountMap[allActorMsgsCount.ActorID] += allActorMsgsCount.Count
		}

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshActorTransferMsgs...")
		allActorsTransferMsgsCount, err := GetAllActorsTransferMsgsCount(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
		if err != nil {
			return err
		}

		for _, allActorTransferMsgsCount := range allActorsTransferMsgsCount {
			dbState.ActorTransfersCountMap[allActorTransferMsgsCount.ActorID] += allActorTransferMsgsCount.Count
		}

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshMinedMsgsMaps...")
		allMinersMinedCount, err := GetAllMinersMinedCount(ctx, dbState.StartEpoch, dbState.EndEpoch, cols)
		if err != nil {
			return err
		}

		for _, allMinerMinedCount := range allMinersMinedCount {
			dbState.MinedMsgsMap[allMinerMinedCount.ActorID] += allMinerMinedCount.Count
		}

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		log.Infow("begin RefreshTransfersForLargeAmount...")
		transfersLargeAmountCount, err := RefreshTransfersForLargeAmount(ctx, dbState, cols, "", "")
		if err != nil {
			return err
		}

		dbState.TransfersLargeAmountCount = transfersLargeAmountCount

		if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
			log.Errorf("set DataBaseState failed: %v", err)
			return err
		}

		return nil
	}

	if err := dbsm.Stm.SetDataBaseState(newDB.Url(), *dbState); err != nil {
		log.Errorf("set DataBaseState failed: %v", err)
		return err
	}

	return nil
}

//func RefreshDataBaseState(ctx context.Context, dbState *DataBaseState, cols Collections) error {
//	if err := RefreshBlockMsgs(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshBlockMsgsByMethodName(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorMsgsByMethodName(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorMsgs(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshActorTransferMsgs(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshMinedMsgsMaps(ctx, dbState, cols); err != nil {
//		return err
//	}
//	if err := RefreshTransfersForLargeAmount(ctx, dbState, cols); err != nil {
//		return err
//	}
//
//	return nil
//}
