package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dtynn/dix"
	amt4 "github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"github.com/ipfs-force-community/londobell/dep"
)

var replayCmd = &cli.Command{
	Name:  "replay",
	Usage: "replay chain data to repo",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:     "start-height",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "end-ts",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "dsn",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		var components struct {
			fx.In
			CS  *store.ChainStore
			Stm *stmgr.StateManager
		}

		stopper, err := dix.New(ctx,
			Inject(cctx, &components),
		)

		defer stopper(ctx) //nolint:errcheck

		if err != nil {
			return err
		}

		components.CS.StoreEvents(true)

		log.Infof("IsStoringEvents: %v", components.Stm.ChainStore().IsStoringEvents())

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		eventsRootCol := db.Collection("EventsRoot")

		cids, err := lcli.ParseTipSetString(cctx.String("end-ts"))
		if err != nil {
			return err
		}

		ts, err := components.CS.LoadTipSet(ctx, types.NewTipSetKey(cids...))
		if err != nil {
			return fmt.Errorf("importing chain failed: %w", err)
		}

		start := cctx.Int("start-height")
		log.Info("start, end height", start, ts.Height())

		tss := []*types.TipSet{}

		for ts.Height() >= abi.ChainEpoch(start) {
			tss = append(tss, ts)
			ts, err = components.CS.LoadTipSet(ctx, ts.Parents())
			if err != nil {
				return fmt.Errorf("load ts failed: %w", err)
			}
		}

		got := len(tss)
		for i := 0; i < got/2; i++ {
			j := got - i - 1
			tss[i], tss[j] = tss[j], tss[i]
		}

		lim := limiter.New(16)
		var ewg multierror.Group

		for _, ts := range tss {
			ts := ts
			starttime := time.Now()

			ewg.Go(func() error {
				if !lim.Acquire(context.TODO()) {
					return nil
				}

				defer func() {
					lim.Release(context.TODO())
				}()

				cmsgs, err := components.CS.MessagesForTipset(ctx, ts)
				if err != nil {
					log.Error(err)
					return err
				}

				var exist = false
				for _, cmsg := range cmsgs {
					if smsg, ok := cmsg.(*types.SignedMessage); ok {
						if smsg.Signature.Type == crypto.SigTypeDelegated {
							exist = true
							break
						}
					}
				}

				if exist {
					_, ires, err := components.Stm.ExecutionTrace(ctx, ts)
					if err != nil {
						log.Error(err)
						return err
					}

					log.Infof("execute tipset %v, len(ires): %v", ts.Height(), len(ires))

					var eventsRes []*model.EventsRoot
					for _, r := range ires {
						if r.MsgRct != nil && r.MsgRct.Version() == types.MessageReceiptV1 {
							eventsRoot := r.MsgRct.EventsRoot
							if eventsRoot != nil {
								events, err := LoadEvents(ctx, components.CS, *eventsRoot)
								if err != nil {
									log.Errorf("load events for root %v failed: %v", r.MsgRct.EventsRoot.String(), err)
									return err
								}

								etm, err := model.NewEventsRoot(*eventsRoot, events, ts.Height())
								if err != nil {
									log.Errorw("convert to model.EventsRoot", "eventsRoot", eventsRoot, "mcid", r.Msg.Cid(), "err", err.Error())
								} else {
									eventsRes = append(eventsRes, etm)
								}
							}
						}
					}

					// insert
					var docs []interface{}
					for _, e := range eventsRes {
						docs = append(docs, e)
					}

					total := len(docs)
					if total > 0 {
						ires, err := eventsRootCol.InsertMany(context.TODO(), docs, options.InsertMany().SetOrdered(false))
						if err != nil {
							if actualErr := mgoutil.ExtractActualMgoErrors(err); actualErr != nil {
								return actualErr
							}
						}

						log.Infof("ts %v inserted: %v/%v, elapsed: %v\n", ts.Height(), len(ires.InsertedIDs), total, time.Now().Sub(starttime).String())
						return nil
					}
				}

				return nil
			})
		}

		if err := ewg.Wait(); err != nil {
			return fmt.Errorf("extract part: %w", err)
		}

		return nil
	},
}

func Inject(cctx *cli.Context, target ...interface{}) dix.Option {
	return dix.Options(
		dix.If(len(target) > 0, dix.Populate(5, target...)),
		dep.ContextModule(context.Background()),
		dep.StateManager(),
		dep.InjectChainRepo(cctx), dep.OfflineDataSource(),
		//dep.InjectFullNode(cctx), dep.OnlineDataSource(),
	)
}

func LoadEvents(ctx context.Context, cs *store.ChainStore, eventsRoot cid.Cid) ([]types.Event, error) {
	store := cs.ActorStore(ctx)
	amt, err := amt4.LoadAMT(ctx, store, eventsRoot, amt4.UseTreeBitWidth(types.EventAMTBitwidth))
	if err != nil {
		return nil, err
	}

	ret := make([]types.Event, 0, amt.Len())
	err = amt.ForEach(ctx, func(u uint64, deferred *cbg.Deferred) error {
		var evt types.Event
		if err := evt.UnmarshalCBOR(bytes.NewReader(deferred.Raw)); err != nil {
			return err
		}
		ret = append(ret, evt)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return ret, nil
}
