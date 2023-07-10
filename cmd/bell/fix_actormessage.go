package main

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/filecoin-project/go-address"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/lib/limiter"
)

//只看块消息
//给定一段高度，查出该高度内的所有消息cid和块cid，和数据库比较

var fixActorMessageCmd = &cli.Command{
	Name:  "fix-actormessage",
	Usage: "fix actorID for ActorMessage",
	Flags: []cli.Flag{
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
		&cli.StringFlag{
			Name:     "nodeconfig",
			Usage:    "The location of the node configuration, eg: ./config.json(node: token)",
			Required: true,
		},
		&cli.DurationFlag{
			Name:  "tick",
			Usage: "tick for fix ActorMessage",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := util.ParseNodes(cctx.String("nodeconfig")); err != nil {
			return err
		}

		fullnode.API = fullnode.NewAppropriateAPI(util.Nodes)
		err := fullnode.API.Choose(ctx)
		if err != nil {
			return err
		}

		api := fullnode.API.GetAppropriateAPI()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		actorMessageCol := db.Collection("ActorMessage")
		tipsetCol := db.Collection("Tipset")
		finalHeightCol := db.Collection("FinalHeight")

		var duration time.Duration
		if !cctx.IsSet("tick") {
			duration = 24 * time.Hour
		} else {
			duration = cctx.Duration("tick")
		}

		tick := time.NewTicker(duration)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				cursor, err := tipsetCol.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}), options.Find().SetLimit(-1))
				if err != nil {
					log.Errorf("tipset find failed: %v", err)
					continue
				}

				var tipsetRes []EpochRes
				if err = cursor.All(context.TODO(), &tipsetRes); err != nil {
					log.Errorf("cur all failed: %v", err)
					continue
				}

				if len(tipsetRes) != 1 {
					log.Warn("not found in Tipset")
					continue
				}

				startEpoch := tipsetRes[0].Epoch

				cursor, err = finalHeightCol.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}), options.Find().SetLimit(-1))
				if err != nil {
					log.Errorf("FinalHeight find failed: %v", err)
					continue
				}

				var finalHeightRes []EpochRes
				if err = cursor.All(context.TODO(), &finalHeightRes); err != nil {
					log.Errorf("cur all failed: %v", err)
					continue
				}

				if len(finalHeightRes) != 1 {
					log.Warn("not found in FinalHeight")
					continue
				}

				endEpoch := finalHeightRes[0].Epoch

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

				var (
					notFoundIDs = make(map[address.Address]struct{})
					nlk         sync.RWMutex
				)

				lim := limiter.New(16)
				var ewg multierror.Group

				log.Infof("range:[%v, %v] begin fix ActorID for ActorMessage", startEpoch, endEpoch)
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

						pipe, err := util.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, getToFixActorMessageJS)
						if err != nil {
							return err
						}

						cur, err := actorMessageCol.Aggregate(context.TODO(), pipe)
						if err != nil {
							return err
						}

						var res []ActorMessageRes
						err = cur.All(context.TODO(), &res)
						if err != nil {
							return err
						}

						var toCompleteMap = make(map[string]address.Address)
						for _, r := range res {
							toAddr, err := address.NewFromString(buildnet.NetPrefix + r.ActorID)
							if err != nil {
								return err
							}

							nlk.RLock()
							_, ok := notFoundIDs[toAddr]
							nlk.RUnlock()
							if ok {
								continue
							}

							toID, err := api.StateLookupID(ctx, toAddr, types.EmptyTSK)
							if err != nil {
								nlk.Lock()
								notFoundIDs[toAddr] = struct{}{}
								nlk.Unlock()

								log.Errorf("lookup ID for %v failed: %v", toAddr, err)
								continue
							}

							toCompleteMap[r.ID] = toID
						}

						for id, actorID := range toCompleteMap {
							pipe, err := util.Parse(model.Ctx{PrimaryID: id, Addr: actorID.String()[1:]}, mergeFixActorMessageJS)
							if err != nil {
								return err
							}

							cur, err := actorMessageCol.Aggregate(ctx, pipe)
							if err != nil {
								return err
							}

							var res []bson.M
							err = cur.All(ctx, &res)
							if err != nil {
								return err
							}
						}

						log.Infof("part [%v, %v] complete: %v, elapsed: %v\n", r.Start, r.End, len(toCompleteMap), time.Now().Sub(starttime).String())
						return nil
					})
				}

				if err := ewg.Wait(); err != nil {
					log.Errorf("falied: %v", err)
					continue
				}

				log.Infof("all finished, [%v, %v] elapsed: %v\n", startEpoch, endEpoch, time.Now().Sub(starttime).String())

			case <-ctx.Done():
				log.Infof("ctx done!!")
			}
		}
	},
}

var getToFixActorMessageJS = "[\n    {\n        $match: {\n            Type: \"to\",\n            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},\n            ActorID: {$not: /^0/}\n        }\n    }\n]"

var mergeFixActorMessageJS = "[\n    {\n        $match: {\n            _id: ctx.PrimaryID,\n        }\n    },\n    {\n        $addFields: {\n            ActorID: ctx.Addr\n        }\n    },\n    {\n        $project: {\n            _id: 1,\n            ActorID: \"$ActorID\"\n        }\n    },\n    {\n        $merge: {\n            into: \"ActorMessage\",\n            on: \"_id\",\n            whenMatched:   \"merge\",\n            whenNotMatched: \"discard\"\n        }\n    }\n]"

type ActorMessageRes struct {
	ID      string `bson:"_id"`
	ActorID string
}

type EpochRes struct {
	Epoch int64 `bson:"_id"`
}
