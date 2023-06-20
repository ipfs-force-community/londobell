package main

import (
	"context"
	"io/ioutil"
	"sort"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/go-state-types/big"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/ipfs-force-community/londobell/buildnet"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/go-address"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/lib/limiter"
)

var completeActorMessageCmd = &cli.Command{
	Name: "complete-actormessage",
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
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "nodeconfig",
			Usage:    "The location of the node configuration, eg: ./config.json(api: token)",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		if err := util.ParseNodes(cctx.String("nodeconfig")); err != nil {
			return err
		}

		adapter.API = adapter.NewAppropriateAPI(util.Nodes)
		err := adapter.API.Choose(ctx)
		if err != nil {
			return err
		}

		api := adapter.API.GetAppropriateAPI()

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		actorMessageCol := db.Collection("ActorMessage")
		js, err := ioutil.ReadFile("./cmd/bell/merge_actormessage.js")
		if err != nil {
			return err
		}

		startEpoch, endEpoch := cctx.Int64("start"), cctx.Int64("end")
		tsDone := startEpoch

		type Range struct {
			Start int64
			End   int64
		}

		part := make([]Range, 0)
		for tsDone < endEpoch {
			start := tsDone
			end := start + 2880*1
			if end > endEpoch {
				end = endEpoch
			}

			part = append(part, Range{Start: start, End: end})

			tsDone = end
		}

		sort.Slice(part, func(i, j int) bool {
			return part[i].Start > part[j].Start
		})

		lim := limiter.New(2)
		var ewg multierror.Group

		log.Infof("begin complete actormessage")
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

				pipe, err := aggregators.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, string(js))
				if err != nil {
					return err
				}

				cur, err := traceCol.Aggregate(context.TODO(), pipe)
				if err != nil {
					return err
				}

				var res []ActorMessageRes
				err = cur.All(context.TODO(), &res)
				if err != nil {
					return err
				}

				actorMessages := make([]ActorMessage, 0)
				for _, r := range res {
					storeMap := make(map[address.Address]string)

					fromAddr, err := address.NewFromString(buildnet.NetPrefix + r.From)
					if err != nil {
						return err
					}

					fromID, err := api.StateLookupID(ctx, fromAddr, types.EmptyTSK)
					if err != nil {
						fromID = fromAddr
					}

					storeMap[fromID] = "from"

					toAddr, err := address.NewFromString(buildnet.NetPrefix + r.To)
					if err != nil {
						return err
					}

					toID, err := api.StateLookupID(ctx, toAddr, types.EmptyTSK)
					if err != nil {
						toID = toAddr
					}

					if _, ok := storeMap[toID]; !ok {
						storeMap[toID] = "to"
					}

					c, err := cid.Decode(r.Cid)
					if err != nil {
						return err
					}

					var sc cid.Cid
					sc, _ = cid.Decode(r.SignedCid)

					v, err := big.FromString(r.Value)
					if err != nil {
						return err
					}
					for actorID, mtype := range storeMap {
						// todo: insert进去类型是否一致
						actorMessages = append(actorMessages, ActorMessage{ID: r.ID + "-" + mtype,
							ActorID:    actorID,
							Epoch:      abi.ChainEpoch(r.Epoch),
							Cid:        c,
							SignedCid:  sc, // null?
							Value:      v,
							MethodName: r.MethodName,
							ExitCode:   exitcode.ExitCode(r.ExitCode),
							From:       fromAddr,
							To:         toAddr,
							IsBlock:    r.IsBlock,
							Type:       mtype})
					}
				}

				var docs []interface{}
				for _, am := range actorMessages {
					docs = append(docs, am)
				}

				total := len(docs)
				if total > 0 {
					ires, err := actorMessageCol.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
					if err != nil {
						if actualErr := extractActualMgoErrors(err); actualErr != nil {
							return actualErr
						}
					}

					log.Infof("part [%v, %v], inserted: %v/%v, elapsed: %v\n", r.Start, r.End, len(ires.InsertedIDs), total, time.Now().Sub(starttime).String())
					return nil
				}

				log.Infof("part [%v, %v] total 0, elapsed: %v\n", r.Start, r.End, time.Now().Sub(starttime).String())
				return nil
			})
		}

		//	2500422-2849164
		//[{2500422 2586822} {2586822 2673222} {2673222 2759622} {2759622 2846022} {2846022 2849164}]
		//{2500422 2514822} {2514822 2529222} {2529222 2543622} {2543622 2558022} {2558022 2572422} {2572422 2586822} {2586822 2601222} {2601222 2615622} {2615622 2630022} {2630022 2644422} {2644422 2658822} ...
		if err := ewg.Wait(); err != nil {
			log.Errorf("falied: %v", err)
			return err
		}

		log.Infof("all finished, elapsed: %v\n", time.Now().Sub(starttime).String())

		return nil
	},
}

type ActorMessageRes struct {
	ID         string `bson:"_id"`
	Epoch      int64
	Cid        string
	SignedCid  string
	Value      string
	MethodName string
	ExitCode   int64
	From       string
	To         string
	IsBlock    bool
}

type ActorMessage struct {
	ID         string `bson:"_id"`
	ActorID    address.Address
	Epoch      abi.ChainEpoch
	Cid        cid.Cid
	SignedCid  cid.Cid
	Value      abi.TokenAmount
	MethodName string
	ExitCode   exitcode.ExitCode
	From       address.Address
	To         address.Address
	IsBlock    bool
	Type       string
}
