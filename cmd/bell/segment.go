package main

import (
	"fmt"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/api/v0api"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/racailum/segment"
)

var segmentCmd = &cli.Command{
	Name: "segment",
	Subcommands: []*cli.Command{
		segmentUpdateCmd,
		segmentShowCmd,
	},
}

var catchMinerInfoCmd = &cli.Command{
	Name: "catch",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "dsn",
		},
		&cli.Int64Flag{
			Name: "start",
		},
		&cli.Int64Flag{
			Name: "end",
		},
	},
	Action: func(cctx *cli.Context) error {
		mutil, err := mgoutil.Connect(cctx.Context, cctx.String("dsn"))
		if err != nil {
			return err
		}
		col := mutil.Database("bell").Collection("ExecTrace")
		start := cctx.Int64("start")
		end := cctx.Int64("end")
		result := make([]bson.M, 0)
		for i := start; i < end; i += 2880 {
			var wdpost, pre, prove, agg, preb float64
			pipe, err := util.Parse(
				model.Ctx{Start: i,
					End:      i + 2880,
					CurEpoch: 5,
				}, p)
			if err != nil {
				return err
			}
			cur, err := col.Aggregate(cctx.Context, pipe)
			if err != nil {
				return err
			}
			err = cur.All(cctx.Context, &result)
			if err != nil {
				return err
			}
			wdpost = result[0]["total_c"].(float64)

			pipe, err = util.Parse(
				model.Ctx{Start: i,
					End:      i + 2880,
					CurEpoch: 6,
				}, p)
			if err != nil {
				return err
			}
			cur, err = col.Aggregate(cctx.Context, pipe)
			if err != nil {
				return err
			}
			err = cur.All(cctx.Context, &result)
			if err != nil {
				return err
			}
			pre = result[0]["total_c"].(float64)

			pipe, err = util.Parse(
				model.Ctx{Start: i,
					End:      i + 2880,
					CurEpoch: 7,
				}, p)
			if err != nil {
				return err
			}
			cur, err = col.Aggregate(cctx.Context, pipe)
			if err != nil {
				return err
			}
			err = cur.All(cctx.Context, &result)
			if err != nil {
				return err
			}
			prove = result[0]["total_c"].(float64)

			pipe, err = util.Parse(
				model.Ctx{Start: i,
					End:      i + 2880,
					CurEpoch: 25,
				}, p)
			if err != nil {
				return err
			}
			cur, err = col.Aggregate(cctx.Context, pipe)
			if err != nil {
				return err
			}
			err = cur.All(cctx.Context, &result)
			if err != nil {
				return err
			}
			preb = result[0]["total_c"].(float64)

			pipe, err = util.Parse(
				model.Ctx{Start: i,
					End:      i + 2880,
					CurEpoch: 26,
				}, p)
			if err != nil {
				return err
			}
			cur, err = col.Aggregate(cctx.Context, pipe)
			if err != nil {
				return err
			}
			err = cur.All(cctx.Context, &result)
			if err != nil {
				return err
			}
			agg = result[0]["total_c"].(float64)

			fmt.Printf("%.4f,%.4f,%.4f,%.4f,%.4f\n", wdpost/1e18, pre/1e18, prove/1e18, preb/1e18, agg/1e18)
		}
		return nil
	},
}
var p = `[{$match :{ Depth: 1, "MsgRct.ExitCode": 0,"Msg.Method": ctx.CurEpoch, "Msg.To":"01228108",Epoch: {$gte: ctx.Start, $lt: ctx.End}}},{$addFields: {converted_c: {$convert: {input: "$GasCost.TotalCost", to : "double", onError: 0, onNull: 0}}}},{$group:{_id:null, total_c: { $sum: "$converted_c" }}}]`
var segmentUpdateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.StringFlag{
			Name:   "dsn-write",
			Hidden: true,
		},

		&cli.StringSliceFlag{
			Name: "dsn-write-slice",
		},

		&cli.StringFlag{
			Name: "dsn-read",
		},

		&cli.StringFlag{
			Name: "child-hi",
		},

		&cli.StringFlag{
			Name: "child-lo",
		},

		&cli.BoolFlag{
			Name: "set-active",
		},
	},

	Action: func(cctx *cli.Context) error {
		di := struct {
			fx.In
			SegMgr *segment.Manager
			Full   v0api.FullNode
			CStore common.ChainStore
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &di),
			dep.InjectRepoPath(cctx),
			dep.InjectFullNode(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		segname := cctx.String("name")
		slog := log.With("seg", segname)

		setInfo := false
		info, _, err := di.SegMgr.LoadInfo(segname)
		if err != nil {
			return err
		}

		if dsn := cctx.String("dsn-write"); dsn != "" {
			setInfo = true
			info.DSN.Write = dsn
		}

		//多写数据库
		if dsnSlice := cctx.StringSlice("dsn-write-slice"); len(dsnSlice) != 0 {
			setInfo = true
			info.DSN.NewWrite = dsnSlice
		}

		if cctx.IsSet("dsn-read") {
			setInfo = true
			info.DSN.Read = cctx.String("dsn-read")
		}

		if setInfo {
			if err := di.SegMgr.SetInfo(segname, info); err != nil {
				return err
			}

			slog.Infof("info set: %#v", info)
		}

		if cctx.IsSet("child-hi") || cctx.IsSet("child-lo") {
			if err := setSegmentBoundary(cctx, slog, di.Full, di.CStore, segname, di.SegMgr); err != nil {
				return err
			}
		}

		if cctx.Bool("set-active") {
			if err := di.SegMgr.SetActive(segname); err != nil {
				return err
			}

			slog.Info("activated")
		}

		slog.Info("done")
		return nil
	},
}

func setSegmentBoundary(cctx *cli.Context, slog *zap.SugaredLogger, full v0api.FullNode, cstore common.ChainStore, segname string, segmgr *segment.Manager) error {
	bound, _, err := segmgr.LoadBoundary(segname)
	if err != nil {
		return fmt.Errorf("load boundary for %s: %w", segname, err)
	}

	set := false
	if s := cctx.String("child-hi"); s != "" {
		tsk, err := parsetTipSetKey(s)
		if err != nil {
			return fmt.Errorf("hi-child key: %w", err)
		}

		hi, err := common.LoadLinkedTipSet(cstore, tsk)
		if err != nil {
			return fmt.Errorf("load hi tipset: %w", err)
		}

		set = true
		bound.SetHi(hi)
	}

	if s := cctx.String("child-lo"); s != "" {
		tsk, err := parsetTipSetKey(s)
		if err != nil {
			return fmt.Errorf("lo-child key: %w", err)
		}

		lo, err := common.LoadLinkedTipSet(cstore, tsk)
		if err != nil {
			return fmt.Errorf("load lo tipset: %w", err)
		}

		set = true
		bound.SetLo(lo)
	}

	if set {
		if bound.Hi.Epoch < bound.Lo.Epoch {
			return fmt.Errorf("hi-epoch(%d) lower than lo-epoch(%d)", bound.Hi.Epoch, bound.Lo.Epoch)
		}

		if err := segmgr.SetBoundary(segname, bound); err != nil {
			return err
		}

		slog.Infow("boundary set", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
	}

	return nil
}

var segmentShowCmd = &cli.Command{
	Name: "show",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},

		&cli.BoolFlag{
			Name:  "info",
			Value: true,
		},

		&cli.BoolFlag{
			Name:  "boundary",
			Value: true,
		},

		&cli.BoolFlag{
			Name:  "active",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		api, _, err := GetAPIV0(cctx)
		if err != nil {
			return err
		}
		segname := cctx.String("name")
		slog := log.With("seg", segname)
		detail, err := api.SegmentDetail(cctx.Context, segname)
		if err != nil {
			return err
		}
		if cctx.Bool("info") {
			if detail.Info != nil {
				slog.Infow("info", "dns-write-slice", detail.Info.DSN.NewWrite, "dns-read", detail.Info.DSN.Read)
			} else {
				slog.Info("segment info not found")
			}
		}
		if cctx.Bool("boundary") {
			if detail.Boundary != nil {
				bound := detail.Boundary
				slog.Infow("boundary", "hi-epoch", bound.Hi.Epoch, "hi-tsk", bound.Hi.TSK.String(), "lo-epoch", bound.Lo.Epoch, "lo-tsk", bound.Lo.TSK)
			} else {
				slog.Info("segment boundary not found")
			}
		}
		if cctx.Bool("active") {
			if detail.Active == "" {
				slog.Infof("active segment not found")
			} else {
				slog.Infof("current active: %s", detail.Active)
			}
		}
		return nil
	},
}
