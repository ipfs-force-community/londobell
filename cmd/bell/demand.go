package main

import (
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/ipfs-force-community/londobell/lib/mgoutil"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/urfave/cli/v2"
)

var demandCmd = &cli.Command{
	Name: "demand",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "dsn",
		},
		&cli.StringFlag{
			Name: "name",
		},
	},
	Action: func(cctx *cli.Context) error {
		rcli, err := mgoutil.Connect(cctx.Context, cctx.String("dsn"))
		if err != nil {
			return err
		}

		database := rcli.Database(cctx.String("name"))
		minerFundsCol := database.Collection("MinerFunds")
		execTraceCol := database.Collection("ExecTrace")

		full, closer, err := client.NewFullNodeRPCV0(cctx.Context, cctx.String("api-url"), nil)
		if err != nil {
			return err
		}
		defer closer()

		ts, err := full.ChainHead(context.TODO())
		if err != nil {
			return err
		}

		miners, err := full.StateListMiners(context.TODO(), ts.Key())
		if err != nil {
			return err
		}

		log.Infof("ts: %v all miners: %v", ts.Height(), len(miners))

		type Res struct {
			Miners []string
		}

		var (
			res  Res
			res2 Res
			res3 Res
		)

		latestHeight := int64(3088200)
		pipe, err := util.Parse(model.Ctx{StartEpoch: latestHeight}, allActiveMinerJS)
		if err != nil {
			return err
		}

		cur, err := minerFundsCol.Aggregate(context.TODO(), pipe)
		if err != nil {
			return err
		}

		err = cur.All(context.TODO(), &res)
		if err != nil {
			return err
		}

		err = WriteToExcelAllActiveMiner(res.Miners)
		if err != nil {
			return err
		}

		pipe2, err := util.Parse(model.Ctx{StartEpoch: 3000240, EndEpoch: 3086640}, newActiveMinerJS)
		if err != nil {
			return err
		}

		cur2, err := execTraceCol.Aggregate(context.TODO(), pipe2)
		if err != nil {
			return err
		}

		err = cur2.All(context.TODO(), &res2)
		if err != nil {
			return err
		}

		err = WriteToExcelAllActiveMiner2(res2.Miners)
		if err != nil {
			return err
		}

		pipe3, err := util.Parse(model.Ctx{StartEpoch: 2913840, EndEpoch: 3000240}, newActiveMinerJS)
		if err != nil {
			return err
		}

		cur3, err := execTraceCol.Aggregate(context.TODO(), pipe3)
		if err != nil {
			return err
		}

		err = cur3.All(context.TODO(), &res3)
		if err != nil {
			return err
		}

		err = WriteToExcelAllActiveMiner3(res3.Miners)
		if err != nil {
			return err
		}

		return nil
	},
}

var allActiveMinerJS = "[\n    {\n        $match: {\n            Epoch: ctx.StartEpoch\n        }\n    },\n    {\n        $group: {\n            _id: 0,\n            Miners: {$addToSet: \"$Addr\"}\n        }\n    }\n]"

var newActiveMinerJS = "[\n    {\n        $match: {\n            IsBlock: true,\n            \"Msg.MethodName\": \"CreateMiner\",\n            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},\n\n        }\n    },\n    {\n        $group: {\n            _id: 0,\n            Miners: {$addToSet: \"$Detail.Return.IDAddress\"}\n        }\n    }\n]"

func WriteToExcelAllActiveMiner(activeMiners []string) error {
	f := excelize.NewFile()
	for i := 0; i < len(activeMiners); i++ {
		if i == 0 {
			if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+1), "miner"); err != nil {
				return err
			}
		}
		if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), activeMiners[i]); err != nil {
			return err
		}
	}

	if err := f.SaveAs("excel/activeminer.xlsx"); err != nil {
		return err
	}

	return nil
}

func WriteToExcelAllActiveMiner2(activeMiners []string) error {
	f := excelize.NewFile()
	for i := 0; i < len(activeMiners); i++ {
		if i == 0 {
			if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+1), "miner"); err != nil {
				return err
			}
		}
		if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), activeMiners[i]); err != nil {
			return err
		}
	}

	if err := f.SaveAs("excel/activeminer2.xlsx"); err != nil {
		return err
	}

	return nil
}

func WriteToExcelAllActiveMiner3(activeMiners []string) error {
	f := excelize.NewFile()
	for i := 0; i < len(activeMiners); i++ {
		if i == 0 {
			if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+1), "miner"); err != nil {
				return err
			}
		}
		if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), activeMiners[i]); err != nil {
			return err
		}
	}

	if err := f.SaveAs("excel/activeminer1.xlsx"); err != nil {
		return err
	}

	return nil
}
