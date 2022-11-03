package main

import (
	"context"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
)

var checkCmd = &cli.Command{
	Name:  "check",
	Usage: "check database data integrity",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.IntFlag{
			Name:  "old",
			Usage: "specify the old database index when len(NewWrite) > 1, starting from zero; -1 when len(NewWrite) == 1",
			Value: -1,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx)
		if err != nil {
			return err
		}

		segName := cctx.String("name")
		clog := log.With("seg", segName)

		var (
			dsns          []string
			loEpoch       abi.ChainEpoch
			enableTracing bool
			interval      abi.ChainEpoch
		)

		detail, err := api.SegmentDetail(cctx.Context, segName)
		if err != nil {
			clog.Errorf("get segment detail: %v", err)
			return err
		}

		if detail.Info != nil {
			dsns = detail.Info.DSN.NewWrite
			clog.Infow("info", "dns-write-slice", dsns)
		} else {
			clog.Error("segment Info not found")
			return nil
		}

		if detail.Boundary != nil {
			loEpoch = detail.Boundary.Lo.Epoch
		} else {
			clog.Error("segment Boundary not found")
			return nil
		}

		ticksMap := make(map[string]int)
		zeroHourMap := make(map[string]bool)
		enableExtractMap := make(map[string]bool)

		//读取每张表的配置（抽取间隔、是否抽取）是否零点抽取、child-lo高度
		allTableList, err := getExtractConfig(cctx, &enableTracing, &interval, ticksMap, zeroHourMap, enableExtractMap)
		if err != nil {
			clog.Errorf("get extract config: %v", err)
			return err
		}

		var olddsn string
		haveOld := false
		if len(dsns) < 2 || cctx.Int("old") == -1 {
			clog.Info("No old db to check")
		} else {
			olddsn = dsns[cctx.Int("old")]
			haveOld = true
		}

		//检查旧库、新库数据是否插入
		for index, dsn := range dsns {
			cilog := clog.With("index", index)

			rcli, err := mgoutil.Connect(cctx.Context, dsn)
			if err != nil {
				cilog.Errorf("connect failed: %v", err)
				return err
			}

			database := rcli.Database(segName)

			//基本检查
			hasData := true
			isConsistent := true
			for _, at := range allTableList {
				startEpoch := getStartEpoch(at, interval, loEpoch, ticksMap, zeroHourMap)
				hasData, err = checkHasData(cctx.Context, database, at, enableExtractMap, startEpoch)
				if err != nil {
					cilog.Errorf("checkHasData error: %v", err)
					return err
				}

				if !hasData {
					cilog.Errorf("table %v has no data", at)
					return nil //break
				}

				//检查旧库数据是否连贯
				if dsn == olddsn {
					endEpoch := getLastEndEpoch(startEpoch, at, interval, ticksMap, zeroHourMap)
					//todo:双写前改了旧库的配置咋办
					isConsistent, err = checkHasData(cctx.Context, database, at, enableExtractMap, endEpoch)
					if err != nil {
						cilog.Errorf("olddb checkHasData error: %v", err)
						return err
					}

					if !isConsistent {
						cilog.Errorf("data in olddb table %v is not consistent", at)
						return nil
					}
				}
			}

			cilog.Info("all table has data!")

			//隐式消息检查
			hasData, err = checkHasImplicitMessages(cctx.Context, database, "ExecTrace", loEpoch, enableTracing)
			if err != nil {
				cilog.Errorf("checkHasImplicitMessages error: %v", err)
				return err
			}

			if !hasData {
				cilog.Error("table ExecTrace has no implicit message")
			}

			cilog.Info("table ExecTrace has implicit messages!")

			//检查旧库数据是否连贯
			if haveOld && isConsistent && index == cctx.Int("old") {
				cilog.Info("data in olddb is consistent!")
			}
		}

		return nil
	},
}

func getExtractConfig(cctx *cli.Context, enableTracing *bool, interval *abi.ChainEpoch, ticksMap map[string]int, zeroHourMap, enableExtractMap map[string]bool) ([]string, error) {
	rpath, err := dep.GetRepoPath(cctx)
	if err != nil {
		return nil, err
	}

	config, err := dep.LoadRaConfig(rpath)
	if err != nil {
		return nil, err
	}

	allTableList := make([]string, len(config.Segment.AllToCheckTableList))

	for i, table := range config.Segment.AllToCheckTableList {
		allTableList[i] = table
		if table == "ExecTrace" || table == "Message" || table == "Tipset" {
			ticksMap[table] = 0
		} else {
			ticksMap[table] = 1
		}

		zeroHourMap[table] = false
		enableExtractMap[table] = config.Segment.Extract.ExtractOptions.EnabelExtract.EnableExtractState
	}

	//反射
	stateRegularType := reflect.TypeOf(config.Segment.Extract.ExtractOptions.StateRegular)
	stateRegularValue := reflect.ValueOf(config.Segment.Extract.ExtractOptions.StateRegular)
	for i := 0; i < stateRegularType.NumField(); i++ {
		name := stateRegularType.Field(i).Name
		if name == "Interval" {
			continue
		}

		name = strings.TrimSuffix(name, "Ticks")
		ticksMap[name] = int(stateRegularValue.Field(i).Int())
		if name == "DealProposalDetail" {
			ticksMap["DealProposal"] = int(stateRegularValue.Field(i).Int())
		}
		if name == "MinerSectorSummary" {
			ticksMap["MinerDealSector"] = int(stateRegularValue.Field(i).Int())
		}
	}

	zeroHourExtractType := reflect.TypeOf(config.Segment.Extract.ExtractOptions.ZeroHourExtract)
	zeroHourExtractValue := reflect.ValueOf(config.Segment.Extract.ExtractOptions.ZeroHourExtract)
	for i := 0; i < zeroHourExtractType.NumField(); i++ {
		name := zeroHourExtractType.Field(i).Name
		zeroHourMap[name] = zeroHourExtractValue.Field(i).Bool()
	}

	enabelExtractType := reflect.TypeOf(config.Segment.Extract.ExtractOptions.EnabelExtract)
	enabelExtractValue := reflect.ValueOf(config.Segment.Extract.ExtractOptions.EnabelExtract)
	for i := 0; i < enabelExtractType.NumField(); i++ {
		name := enabelExtractType.Field(i).Name
		if name == "EnableExtractState" {
			continue
		}

		name = strings.TrimPrefix(name, "EnableExtract")
		enableExtractMap[name] = enabelExtractValue.Field(i).Bool()
	}

	*enableTracing = config.EnableTracing
	*interval = config.Segment.Extract.ExtractOptions.StateRegular.Interval

	return allTableList, nil
}

//得到最早入库时间
func getStartEpoch(tableName string, interval abi.ChainEpoch, loEpoch abi.ChainEpoch, ticksMap map[string]int, zeroHourMap map[string]bool) abi.ChainEpoch {
	startEpoch := loEpoch + 1

	isZeroHour := zeroHourMap[tableName]
	intervalT := interval * abi.ChainEpoch(ticksMap[tableName])
	if intervalT == 0 || startEpoch%intervalT == 0 {
		log.Infow("getStartEpoch [intervalT == 0 || startEpoch%intervalT == 0]", "tableName", tableName, "loEpoch", loEpoch, "intervalT", intervalT, "isZeroHour", isZeroHour, "startEpoch", startEpoch)
		return startEpoch
	}

	//loEpoch与整点或零点的距离高度
	var restHeight, restHeight1, restHeight2 = 0, 0, 0

	restHeight1 = int(intervalT - startEpoch%intervalT)

	if isZeroHour {
		curTime := time.Unix(common.BaseTime.Unix()+int64(startEpoch)*30, 0).In(common.Loc)
		diffSec := curTime.Hour()*60*60 + curTime.Minute()*60 + curTime.Second()
		if diffSec == 0 {
			log.Infow("getStartEpoch [zero hour]", "tableName", tableName, "loEpoch", loEpoch, "intervalT", intervalT, "isZeroHour", isZeroHour, "startEpoch", startEpoch)
			return startEpoch
		}
		restHeight2 = 2880 - diffSec/30
	} else {
		restHeight2 = restHeight1
	}

	if restHeight1 != 0 && restHeight2 != 0 {
		restHeight = int(math.Min(float64(restHeight1), float64(restHeight2)))
		startEpoch = startEpoch + abi.ChainEpoch(restHeight)
	}

	log.Infow("getStartEpoch", "tableName", tableName, "loEpoch", loEpoch, "intervalT", intervalT, "isZeroHour", isZeroHour, "startEpoch", startEpoch)

	return startEpoch
}

//基本检查：检查是否有数据
func checkHasData(ctx context.Context, database *mongo.Database, tableName string, enableExtractMap map[string]bool, epoch abi.ChainEpoch) (bool, error) {
	if !enableExtractMap[tableName] {
		log.Infow("checkHasData", "table cannot be extracted", tableName)
		return true, nil
	}

	epochKey := "Epoch"
	switch tableName {
	case "DealProposalDetail", "DealProposalSummary":
		epochKey = "ActorStateExBasic.Epoch"
	case "FilSupply", "Tipset":
		epochKey = "_id"
	case "Message":
		epochKey = "Detail.PackedHeight"
	default:
		epochKey = "Epoch"
	}

	collection := database.Collection(tableName)
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{
					Key: epochKey, Value: bson.D{
						{Key: "$eq", Value: epoch},
					},
				},
			},
		},
	}

	limitStage := bson.D{
		{
			Key: "$limit", Value: 1,
		},
	}

	groupState := bson.D{
		{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "Count", Value: bson.D{
					{Key: "$sum", Value: 1},
				}},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{matchStage, limitStage, groupState})
	if err != nil {
		return false, err
	}

	type result struct {
		Count float64
	}

	var results []result

	if err := cursor.All(ctx, &results); err != nil {
		return false, err
	}

	if len(results) == 1 {
		if results[0].Count == 1 {
			return true, nil
		}
	}

	log.Errorw("checkHasData", "len(results)", len(results))

	return false, nil

}

//enableTracing为true时，检查ExecTrace有隐式消息，即depth>1
func checkHasImplicitMessages(ctx context.Context, database *mongo.Database, tableName string, loEpoch abi.ChainEpoch, enableTracing bool) (bool, error) {
	if !enableTracing {
		return true, nil
	}

	startEpoch := loEpoch + 1
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{
					Key: "Epoch", Value: bson.D{
						{Key: "$eq", Value: startEpoch},
					},
				},
				{
					Key: "Depth", Value: bson.D{
						{Key: "$gt", Value: 1},
					},
				},
			},
		},
	}

	limitStage := bson.D{
		{
			Key: "$limit", Value: 1,
		},
	}

	groupState := bson.D{
		{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "Count", Value: bson.D{
					{Key: "$sum", Value: 1},
				}},
			},
		},
	}

	collection := database.Collection(tableName)

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{matchStage, limitStage, groupState})
	if err != nil {
		return false, err
	}

	type result struct {
		Count float64
	}

	var results []result

	if err := cursor.All(ctx, &results); err != nil {
		return false, err
	}

	if len(results) == 1 {
		if results[0].Count == 1 {
			return true, nil
		}
	}

	return false, nil
}

//加上双写数据库前，旧库保证数据连贯的最高抽取高度
func getLastEndEpoch(startEpoch abi.ChainEpoch, tableName string, interval abi.ChainEpoch, ticksMap map[string]int, zeroHourMap map[string]bool) abi.ChainEpoch {
	endEpoch := startEpoch - 1

	isZeroHour := zeroHourMap[tableName]
	intervalT := interval * abi.ChainEpoch(ticksMap[tableName])
	if intervalT == 0 || endEpoch%intervalT == 0 {
		log.Infow("getLastEndEpoch [intervalT == 0 || endEpoch%intervalT == 0]", "tableName", tableName, "intervalT", intervalT, "isZeroHour", isZeroHour, "endEpoch", endEpoch)
		return endEpoch
	}

	//endEpoch与前一个整点或零点的距离高度
	var endEpoch1, endEpoch2 abi.ChainEpoch

	endEpoch1 = endEpoch - endEpoch%intervalT

	if isZeroHour {
		curTime := time.Unix(common.BaseTime.Unix()+int64(endEpoch)*30, 0).In(common.Loc)
		diffSec := curTime.Hour()*60*60 + curTime.Minute()*60 + curTime.Second()
		if diffSec == 0 {
			log.Infow("getLastEndEpoch [zero hour]", "tableName", tableName, "intervalT", intervalT, "isZeroHour", isZeroHour, "endEpoch", endEpoch)
			return endEpoch
		}
		endEpoch2 = endEpoch - abi.ChainEpoch(diffSec/30)
	} else {
		endEpoch2 = endEpoch1
	}

	endEpoch = abi.ChainEpoch(math.Max(float64(endEpoch1), float64(endEpoch2)))
	log.Infow("getLastEndEpoch", "tableName", tableName, "intervalT", intervalT, "isZeroHour", isZeroHour, "endEpoch", endEpoch)

	return endEpoch
}
