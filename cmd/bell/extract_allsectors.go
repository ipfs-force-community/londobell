package main

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/api/v0api"
	builtin2 "github.com/filecoin-project/lotus/chain/actors/builtin"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

	"github.com/ipfs-force-community/londobell/dep"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/lib/limiter"
)

//只看块消息
//给定一段高度，查出该高度内的所有消息cid和块cid，和数据库比较

var extractAllSectorsCmd = &cli.Command{
	Name:  "extract-allsectors",
	Usage: "extract all sectors at some height",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name: "dsn-write-slice",
		},
		&cli.StringFlag{
			Name:     "ts",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "child",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "limit",
			Required: true,
			Value:    32,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		shutdownCh := make(chan struct{})
		var components struct {
			fx.In
			Full   v0api.FullNode
			CStore common.ChainStore
			Stm    common.StateManager
		}
		_, err := dix.New(ctx,
			dep.WalkRaCalium(cctx, fxlog, &components),
			dep.InjectRepoPath(cctx),
			dep.InjectWritableOffline(cctx),
			dix.Override(new(dtypes.ShutdownChan), shutdownCh),
		)
		if err != nil {
			return err
		}

		tsk, err := parsetTipSetKey(cctx.String("ts"))
		if err != nil {
			return err
		}
		childTsk, err := parsetTipSetKey(cctx.String("child"))
		if err != nil {
			return err
		}

		ts, err := components.CStore.LoadTipSet(ctx, tsk)
		if err != nil {
			return err
		}
		child, err := components.CStore.LoadTipSet(ctx, childTsk)
		if err != nil {
			return err
		}

		height := ts.Height()
		multiWdocs := &mgoutil.MultiDB{}
		for _, write := range cctx.StringSlice("dsn-write-slice") {
			wcli, err := mgoutil.Connect(context.TODO(), write)
			if err != nil {
				return err
			}
			wdb := wcli.Database(cctx.String("name"))
			wdoc, err := mgoutil.NewMgoDocDB(context.TODO(), wcli, wdb)
			if err != nil {
				return err
			}
			err = multiWdocs.SetDbs(wdoc)
			if err != nil {
				return err
			}
		}

		root := child.ParentState()
		tree, err := components.Stm.StateTree(root)
		if err != nil {
			return fmt.Errorf("load state tree for %s: %w", root, err)
		}

		var dal struct {
			common.ChainStore
			common.ChainDict
			common.StateManager
		}
		dal.ChainStore = components.CStore
		dal.StateManager = components.Stm
		elog := log.With("height", height)
		extractCtx, err := extract.NewCtx(ctx, dal, elog, nil, 0, extract.DryOptions())
		if err != nil {
			return err
		}

		astore := components.CStore.ActorStore(ctx)
		allMiners := make(map[address.Address]*types.Actor)
		err = tree.ForEach(func(addr address.Address, act *types.Actor) error {
			if !builtin2.IsStorageMinerActor(act.Code) {
				return nil
			}

			allMiners[addr] = act

			return nil
		})

		if err != nil {
			return fmt.Errorf("walk through actors: %w", err)
		}

		log.Infof("allminers: %v", len(allMiners))

		originCtx := ctx
		innerCtx, innerCancel := context.WithCancel(originCtx)
		defer innerCancel()

		ctx = innerCtx
		defer func() {
			ctx = originCtx
		}()

		var ewg multierror.Group
		var lk sync.Mutex
		lim := limiter.New(cctx.Int("limit"))

		res := extract.NewRes(4096, 0)

		allSectors := []*model.ChangedSector{}
		var totalCount int64
		for addr, act := range allMiners {
			addr := addr
			act := act
			ewg.Go(func() error {
				if !lim.Acquire(innerCtx) {
					return nil
				}

				defer func() {
					lim.Release(innerCtx)
				}()

				select {
				case <-innerCtx.Done():
					return nil

				default:
				}

				var err error
				defer func() {
					if err != nil {
						innerCancel()
					}
				}()

				idaddr, err := extract.LookupID(extractCtx, addr, child)
				if err != nil {
					return fmt.Errorf("lookup ID for addr %v failed: %v", addr, err)
				}

				mas, err := lminer.Load(astore, act)
				if err != nil {
					return fmt.Errorf("load miner state for addr: %v failed: %v", addr, err)
				}

				sectorInfos, err := mas.LoadSectors(nil)
				if err != nil {
					return fmt.Errorf("load all sector infos for addr: %v failed: %v", addr, err)
				}

				for _, sectorInfo := range sectorInfos {
					sector := model.NewChangedSector(*sectorInfo, idaddr, height, false, false)
					lk.Lock()
					allSectors = append(allSectors, sector)
					totalCount++
					lk.Unlock()
				}

				lk.Lock()
				if len(allSectors) > 4096 {
					for i := range allSectors {
						res.Docs = append(res.Docs, allSectors[i])
					}

					docs := make([][]common.Document, 1)
					docs[0] = res.Docs

					elog.Infof("begin insert, count: %v", len(res.Docs))
					if err := insertMany(context.TODO(), elog, docs, multiWdocs); err != nil {
						elog.Errorf("insert failed: %v", err)
						return err
					}

					res.Docs = res.Docs[len(res.Docs):]
					allSectors = allSectors[len(allSectors):]
				}
				lk.Unlock()

				return nil
			})
		}

		if err := ewg.Wait(); err != nil {
			return fmt.Errorf("extract all sectors: %w", err)
		}

		elog.Infof("insert done, totalcount: %v", totalCount)
		return nil

	},
}

func insertMany(ctx context.Context, l *zap.SugaredLogger, docSets [][]common.Document, db common.DocumentDB) error {
	if len(docSets) == 0 {
		return nil
	}

	limit := 4096
	insertDocs := map[string][]interface{}{}
	updateDocs := map[string][]interface{}{}
	insertedCounts := map[string]int{}
	matchedCounts := map[string]int{}
	modifiedCounts := map[string]int{}
	upsertedCounts := map[string]int{}
	totals := map[string]int{}
	insertOps := 0
	updateOps := 0

	insert := func(col string) error {
		if len(insertDocs[col]) == 0 {
			return nil
		}
		var (
			inserted int
			err      error
		)

		insertOps++
		inserted, err = db.Insert(ctx, col, insertDocs[col])
		if err != nil {
			return err
		}

		insertDocs[col] = insertDocs[col][:0]
		insertedCounts[col] = insertedCounts[col] + inserted
		return nil
	}

	for si := range docSets {
		for di := range docSets[si] {
			d := docSets[si][di]
			colName := d.CollectionName()
			if !d.IsMutable() {
				insertDocs[colName] = append(insertDocs[colName], d)
			} else {
				updateDocs[colName] = append(updateDocs[colName], d)
			}

			totals[colName] = totals[colName] + 1

			if len(insertDocs[colName]) >= limit {
				if err := insert(colName); err != nil {
					return err
				}
			}
		}
	}

	for col := range insertDocs {
		if err := insert(col); err != nil {
			return err
		}
	}

	insertColNames := make([]string, 0, len(insertDocs))
	updateColNames := make([]string, 0, len(updateDocs))
	for col := range insertDocs {
		insertColNames = append(insertColNames, col)
	}
	for col := range updateDocs {
		updateColNames = append(updateColNames, col)
	}

	sort.Strings(insertColNames)
	sort.Strings(updateColNames)

	logFields := make([]interface{}, 0, (len(insertColNames)+len(updateColNames))*2+2)
	logFields = append(logFields, "insertOps", insertOps, "updateOps", updateOps)
	for _, col := range insertColNames {
		logFields = append(logFields, col, fmt.Sprintf("%d/%d", insertedCounts[col], totals[col]))
	}

	for _, col := range updateColNames {
		logFields = append(logFields, col, fmt.Sprintf("matched: %v, modified: %d, upserted: %d, total: %d", matchedCounts[col], modifiedCounts[col], upsertedCounts[col], totals[col]))
	}

	l.Infow("documents inserted and updated", logFields...)

	return nil
}
