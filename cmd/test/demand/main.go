package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/limiter"
	logging "github.com/ipfs/go-log/v2"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"reflect"
)

type CountRes struct {
	Count int64
}

type Range struct {
	Start int64
	End   int64
}

type Result struct {
	Date       string
	PowerCount int64
	FvmCount   int64
	AllCount   int64
}

var log = logging.Logger("demand")

func main() {
	// 2/14-5/18
	ctx := context.TODO()
	startEpoch, endEpoch := int64(2599920), int64(2867760)
	parts := make([]Range, 0)
	for epoch := startEpoch; epoch <= endEpoch; epoch++ {
		start := epoch
		end := epoch + 2880
		if end > endEpoch {
			end = endEpoch
		}

		parts = append(parts, Range{Start: start, End: end})
	}

	uri := "mongodb://guest:read-only@dds-uf655172d52c38641.mongodb.rds.aliyuncs.com:3717/bell?replicaSet=mgset-65444697"
	//uri := "mongodb://guest:read-only@dds-uf655172d52c38641732-pub.mongodb.rds.aliyuncs.com:3717/bell?replicaSet=mgset-65444697"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
	if err != nil {
		log.Error(err)
		return
	}
	db := client.Database("bell")
	traceCol := db.Collection("ExecTrace")

	messaesForPowerJs, err := ioutil.ReadFile("./js/messages_for_power.js")
	if err != nil {
		log.Error(err)
		return
	}
	messaesForFVMJs, err := ioutil.ReadFile("./js/messages_for_fvm.js")
	if err != nil {
		log.Error(err)
		return
	}
	messaesForAllJs, err := ioutil.ReadFile("./js/messages_for_all.js")
	if err != nil {
		log.Error(err)
		return
	}

	result := make([]Result, len(parts))
	var ewg multierror.Group
	lim := limiter.New(16)
	for i := range parts {
		i := i
		part := parts[i]
		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				return nil
			}

			defer func() {
				lim.Release(ctx)
			}()

			var (
				powerCount = int64(0)
				fvmCount   = int64(0)
				allCount   = int64(0)
			)
			var powerCountRes []CountRes
			pipe, err := aggregators.Parse(model.Ctx{StartEpoch: part.Start, EndEpoch: part.End}, string(messaesForPowerJs))
			if err != nil {
				return err
			}
			cur, err := traceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}
			err = cur.All(ctx, &powerCountRes)
			if err != nil {
				return err
			}
			if len(powerCountRes) != 0 {
				powerCount = powerCountRes[0].Count
			}

			var fvmCountRes []CountRes
			pipeFVM, err := aggregators.Parse(model.Ctx{StartEpoch: part.Start, EndEpoch: part.End}, string(messaesForFVMJs))
			if err != nil {
				return err
			}
			curFVM, err := traceCol.Aggregate(ctx, pipeFVM)
			if err != nil {
				return err
			}
			err = curFVM.All(ctx, &fvmCountRes)
			if err != nil {
				return err
			}
			if len(fvmCountRes) != 0 {
				fvmCount = fvmCountRes[0].Count
			}

			var allCountRes []CountRes
			pipeAll, err := aggregators.Parse(model.Ctx{StartEpoch: part.Start, EndEpoch: part.End}, string(messaesForAllJs))
			if err != nil {
				return err
			}
			curAll, err := traceCol.Aggregate(ctx, pipeAll)
			if err != nil {
				return err
			}
			err = curAll.All(ctx, &allCountRes)
			if err != nil {
				return err
			}
			if len(allCountRes) != 0 {
				allCount = allCountRes[0].Count
			}

			result[i] = Result{Date: common.CalcTimeByEpoch(uint64(part.Start)).String(), PowerCount: powerCount, FvmCount: fvmCount, AllCount: allCount}
			log.Infof("agg successfully for start: %v, res: %+v", part.Start, Result{Date: common.CalcTimeByEpoch(uint64(part.Start)).String(), PowerCount: powerCount, FvmCount: fvmCount, AllCount: allCount})
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		log.Error("ewg failed: %v", err)
		return
	}

	if err := WriteToExcel(result); err != nil {
		log.Error("WriteToExcel faled: %v", err)
		return
	}
}

func WriteToExcel(res []Result) error {
	f := excelize.NewFile()
	for i := 0; i < len(res); i++ {
		if i == 0 {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+1), "Date")
			f.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+1), "PowerMessagesCount")
			f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+1), "FvmMessagesCount")
			f.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+1), "AllMessagesCount")
		}
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), res[i].Date)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+2), res[i].PowerCount)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+2), res[i].FvmCount)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+2), res[i].AllCount)
	}

	if err := f.SaveAs("excel/messages.xlsx"); err != nil {
		return err
	}

	return nil
}
