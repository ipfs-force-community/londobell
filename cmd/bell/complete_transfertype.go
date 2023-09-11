package main

import (
	"context"
	"sort"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/lib/limiter"
)

var completeTransferTypeCmd = &cli.Command{
	Name:  "complete-transfertype",
	Usage: "complete transfertype for ActorMessage",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "end",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dsn",
			Required: true,
			Usage:    "dsn of database",
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Usage:    "name of database",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		actorMessageCol := db.Collection("ActorMessage")

		startEpoch := cctx.Int64("start")
		endEpoch := cctx.Int64("end")

		tsDone := startEpoch

		type Range struct {
			Start int64
			End   int64
		}

		part := make([]Range, 0)
		for tsDone < endEpoch {
			start := tsDone
			end := start + 2880*5
			if end > endEpoch {
				end = endEpoch
			}

			part = append(part, Range{Start: start, End: end})

			tsDone = end
		}

		sort.Slice(part, func(i, j int) bool {
			return part[i].Start > part[j].Start
		})

		lim := limiter.New(16)
		var ewg multierror.Group

		log.Infof("range:[%v, %v) begin complete transfertype", startEpoch, endEpoch)
		starttime := time.Now()

		for i := range part {
			i := i
			r := part[i]
			ewg.Go(func() error {
				if !lim.Acquire(context.TODO()) {
					return nil
				}

				defer func() {
					lim.Release(context.TODO())
				}()

				pipe, err := util.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, mergeJS)
				if err != nil {
					return err
				}

				cur, err := actorMessageCol.Aggregate(context.TODO(), pipe)
				if err != nil {
					return err
				}

				var res []bson.M
				err = cur.All(context.TODO(), &res)
				if err != nil {
					return err
				}

				log.Infof("part [%v, %v) complete, elapsed: %v\n", r.Start, r.End, time.Now().Sub(starttime).String())
				return nil
			})
		}

		if err := ewg.Wait(); err != nil {
			log.Errorf("falied: %v", err)
			return err
		}

		log.Infof("all finished, [%v, %v) elapsed: %v\n", startEpoch, endEpoch, time.Now().Sub(starttime).String())

		return nil
	},
}

var mergeJS = "[\n    {\n        $match: {\n            \"ExitCode\": 0,\n            \"Value\": {$gt: \"0\"},\n            \"Epoch\": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}\n        }\n    },\n        {\n            $addFields: {\n                TransferType: {$cond: {\n                        if:{\n                            $and: [\n                                {$eq:[\"$Type\", \"to\"]},\n                                {$eq:[\"$From\", \"02\"]},\n                                { $eq:[\"$MethodName\", \"ApplyRewards\"]},\n                                {$gt: [\"$Value\", \"0\"]}\n                            ]\n                        }, then: \"Blockreward\",\n                        else: {\n                            $cond: {\n                                if:{\n                                    $and: [\n                                        {$eq:[\"$Type\", \"from\"]},\n                                        {$eq:[\"$To\", \"099\"]},\n                                        {$gt: [\"$Value\", \"0\"]}\n                                    ]\n                                }, then: \"Burn\",\n                                else: {\n                                    $cond: {\n                                        if: {\n                                            $and: [\n                                                { $eq:[\"$Type\", \"from\"]},\n                                                {$gt: [\"$Value\", \"0\"]},\n                                                { $ne: [\"$To\", \"099\"]}\n                                            ]\n                                        }, then: \"Send\",\n                                        else: {\n                                            $cond: {\n                                                if: {\n                                                    $and: [\n                                                        {$eq:[\"$Type\", \"to\"]},\n                                                        {$gt: [\"$Value\", \"0\"]},\n                                                        {$ne:[\"$MethodName\", \"ApplyRewards\"]}\n                                                    ]\n                                                }, then: \"Receive\",\n                                                else: \"\"\n                                            }\n                                        }\n                                    }\n                                }\n                            }\n                        }\n                    }}\n            }\n        },\n        {\n            $project: {\n                _id: 1,\n                TransferType: \"$TransferType\"\n            }\n        },\n        {\n            $merge: {\n                into: \"ActorMessage\",\n                on: \"_id\",\n                whenMatched:   \"merge\",\n                whenNotMatched: \"discard\"\n            }\n        }\n    ]"
