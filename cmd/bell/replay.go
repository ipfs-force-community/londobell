package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/dtynn/dix"
	amt4 "github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/fx"

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
		//r, err := repo.NewFS(cctx.String("repo"))
		//if err != nil {
		//	return fmt.Errorf("opening fs repo: %w", err)
		//}
		//
		//err = r.Init(repo.FullNode)
		//if err != nil && err != repo.ErrRepoExists {
		//	return fmt.Errorf("repo error: %w", err)
		//}
		//
		//lr, err := r.Lock(repo.FullNode)
		//if err != nil {
		//	return err
		//}
		//defer lr.Close() //nolint:errcheck
		//
		//bs, err := lr.Blockstore(cctx.Context, repo.UniversalBlockstore)
		//if err != nil {
		//	return fmt.Errorf("failed to open blockstore: %w", err)
		//}
		//
		//mds, err := lr.Datastore(context.TODO(), "/metadata")
		//if err != nil {
		//	return err
		//}
		//
		//j, err := fsjournal.OpenFSJournal(lr, journal.EnvDisabledEvents())
		//if err != nil {
		//	return fmt.Errorf("failed to open journal: %w", err)
		//}
		//
		//cst := store.NewChainStore(bs, bs, mds, filcns.Weight, j)

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

		//replay
		for _, ts := range tss {
			_, ires, err := components.Stm.ExecutionTrace(ctx, ts)
			if err != nil {
				log.Error(err)
				return err
			}

			log.Infof("execute tipset %v, len(ires): %v", ts.Height(), len(ires))

			//validate
			for _, r := range ires {
				if r.MsgRct != nil && r.MsgRct.Version() == types.MessageReceiptV1 {
					eventsRoot := r.MsgRct.EventsRoot
					if eventsRoot != nil {
						events, err := LoadEvents(ctx, components.CS, *eventsRoot)
						if err != nil {
							log.Errorf("load events for root %v failed: %v", r.MsgRct.EventsRoot.String(), err)
							return err
						}

						log.Infof("load events for root %v at %v successfly, events: %v", r.MsgRct.EventsRoot.String(), ts.Height(), events)
					}

					log.Infof("null eventsRoot")
				}
			}
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

//func InjectChainRepo(cctx *cli.Context) dix.Option {
//	return dix.Override(new(repo.LockedRepo), func(lc fx.Lifecycle) repo.LockedRepo {
//		r, err := repo.NewFS(cctx.String("repo"))
//		if err != nil {
//			panic(fmt.Errorf("opening fs repo: %w", err))
//		}
//		err = r.Init(repo.FullNode)
//		if err != nil && err != repo.ErrRepoExists {
//			panic(fmt.Errorf("dst repo error: %w", err))
//		}
//
//		lr, err := r.Lock(repo.FullNode)
//		if err != nil {
//			panic(fmt.Errorf("lock repo failed: %w", err))
//		}
//
//		return modules.LockedRepo(lr)(lc)
//	})
//}

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
