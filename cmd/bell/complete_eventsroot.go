package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"sort"
	"time"

	"go.uber.org/fx"

	"github.com/dtynn/dix"

	"github.com/ipfs-force-community/londobell/dep"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/hashicorp/go-multierror"

	"github.com/urfave/cli/v2"

	amt4 "github.com/filecoin-project/go-amt-ipld/v4"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

var completeEventsRootCmd = &cli.Command{
	Name: "complete-eventsroot",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "end",
			Required: true,
			Usage:    "not included",
		},
		&cli.StringFlag{
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "local",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		var components struct {
			fx.In
			CS common.ChainStore
		}

		stopper, err := dix.New(ctx,
			Bell2(cctx, fxlog, cctx.Bool("local"), &components),
			dep.InjectFullNode(cctx),
			dep.InjectRepoPath(cctx),
		)

		defer stopper(ctx) // nolint: errcheck
		if err != nil {
			return err
		}

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		EventsRootCol := db.Collection("EventsRoot")

		js, err := ioutil.ReadFile("./cmd/bell/complete_eventsroot.js")
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

		log.Infof("begin complet eventsroot")
		starttime := time.Now()

		for {
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

					var res []EventsRootRes
					err = cur.All(context.TODO(), &res)
					if err != nil {
						return err
					}

					var eventsRoot = make([]EventsRoot, 0)
					for _, r := range res {
						if r.EventsRoot != "" { // null返回啥
							root, err := cid.Decode(r.EventsRoot)
							if err != nil {
								return err
							}

							// 只同步1月后的
							events, err := GetEvents(context.TODO(), root, components.CS)
							if err != nil {
								return err
							}

							eventsJSON, err := json.Marshal(events)
							if err != nil {
								return err
							}

							eventsRoot = append(eventsRoot, EventsRoot{Cid: r.Cid, Events: eventsJSON, Epoch: r.Epoch})
						} else {
							continue
						}
					}

					// insert into EvmInitCode
					var docs []interface{}
					for _, e := range eventsRoot {
						d := bson.D{
							{Key: "_id", Value: e.Cid},
							{Key: "Events", Value: e.Events},
							{Key: "Epoch", Value: e.Epoch},
						}

						docs = append(docs, d)
					}

					total := len(docs)
					if total > 0 {
						ires, err := EventsRootCol.InsertMany(context.TODO(), docs, options.InsertMany().SetOrdered(false))
						if err != nil {
							if actualErr := extractActualMgoErrors(err); actualErr != nil {
								return actualErr
							}
						}

						log.Infof("part [%v, %v] inserted: %v/%v, elapsed: %v\n", r.Start, r.End, len(ires.InsertedIDs), total, time.Now().Sub(starttime).String())
						return nil
					}

					log.Infof("part [%v, %v] total 0, elapsed: %v\n", r.Start, r.End, time.Now().Sub(starttime).String())
					return nil
				})

			}

			if err := ewg.Wait(); err != nil {
				log.Errorf("falied: %v", err)
				continue
			}

			log.Infof("all finished, elapsed: %v\n", time.Now().Sub(starttime).String())
			break
		}

		return nil
	},
}

type EventsRootRes struct {
	Cid        string
	EventsRoot string
	Epoch      int64
}

type EventsRoot struct {
	Cid    string
	Events []byte
	Epoch  int64
}

func GetEvents(ctx context.Context, root cid.Cid, cs common.ChainStore) ([]types.Event, error) {
	store := cs.ActorStore(ctx)
	evtArr, err := amt4.LoadAMT(ctx, store, root, amt4.UseTreeBitWidth(types.EventAMTBitwidth))
	if err != nil {
		return nil, xerrors.Errorf("load events amt: %w", err)
	}

	ret := make([]types.Event, 0, evtArr.Len())
	var evt types.Event
	err = evtArr.ForEach(ctx, func(u uint64, deferred *cbg.Deferred) error {
		if u > math.MaxInt {
			return xerrors.Errorf("too many events")
		}
		if err := evt.UnmarshalCBOR(bytes.NewReader(deferred.Raw)); err != nil {
			return err
		}

		ret = append(ret, evt)
		return nil
	})

	return ret, err
}

func Bell2(cctx *cli.Context, logger fx.Printer, local bool, target ...interface{}) dix.Option {
	return dix.Options(
		dep.ContextModule(context.Background()),

		dix.If(logger != nil, dix.Logger(logger)),
		dix.If(len(target) > 0, dix.Populate(5, target...)),
		dep.StateManager(),

		dix.ApplyIf(func(s *dix.Settings) bool {
			return local
		}, dep.InjectChainRepo(cctx), dep.OfflineDataSource()),

		dix.ApplyIf(func(s *dix.Settings) bool {
			return !local
		}, dep.InjectFullNode(cctx), dep.OnlineDataSource()),
	)
}
