package main

import (
	"context"
	"fmt"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/dep"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"

	"go.uber.org/fx"

	"github.com/dtynn/dix"
	"github.com/urfave/cli/v2"
)

// segment dsn存在state Info里
var segmentCmd = &cli.Command{
	Name: "segment",
	Subcommands: []*cli.Command{
		segmentUpdateCmd,
		segmentShowCmd,
	},
}

var segmentUpdateCmd = &cli.Command{
	Name: "update",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Usage:    "use the same name for formal and colds",
		},

		&cli.StringFlag{
			Name: "dsn-write",
		},

		&cli.StringFlag{
			Name: "dsn-read",
		},

		&cli.BoolFlag{
			Name: "set-active",
		},
	},

	Action: func(cctx *cli.Context) error {
		di := struct {
			fx.In
			SegMgr *segment.SegManager
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.MultiQuery(context.TODO(), &di),
			dep.InjectRepoPath(cctx),
		)
		if err != nil {
			fmt.Println("stopper", err)
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

		//多写数据库
		//if dsnSlice := cctx.StringSlice("dsn-write"); len(dsnSlice) != 0 {
		//	setInfo = true
		//	info.Writes = dsnSlice
		//}

		if cctx.IsSet("dsn-write") {
			setInfo = true
			info.Write = cctx.String("dsn-write")
		}

		if cctx.IsSet("dsn-read") {
			setInfo = true
			info.Read = cctx.String("dsn-read")
		}

		if setInfo {
			if err := di.SegMgr.SetInfo(segname, info); err != nil {
				return err
			}

			slog.Infof("info set: %#v", info)
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

var segmentShowCmd = &cli.Command{
	Name: "show",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "RPCListen",
			Usage: "multiaddr of rpc",
		},
		&cli.BoolFlag{
			Name:  "local",
			Usage: "load locally if true, otherwise rpc call if false",
		},
	},
	Action: func(cctx *cli.Context) error {
		name := cctx.String("name")
		local := cctx.Bool("local")

		var (
			info  segment.Info
			found bool
		)
		if local {
			di := struct {
				fx.In
				SegMgr *segment.SegManager
			}{}

			stopper, err := dix.New(
				cctx.Context,
				dep.MultiQuery(context.TODO(), &di),
				dep.InjectRepoPath(cctx),
			)
			if err != nil {
				fmt.Println("stopper", err)
				return err
			}

			defer stopper(cctx.Context) // nolint: errcheck

			info, found, err = di.SegMgr.LoadInfo(name)
			if err != nil {
				return err
			}

			if !found {
				log.Warnf("name %v not exist", name)
				return nil
			}
		} else {
			api, _, err := GetAPIV0(cctx.Context, cctx.String("RPCListen"))
			if err != nil {
				return err
			}

			info, err = api.LoadDBInfo(name)
			if err != nil {
				return err
			}
		}

		log.Infof("dbState of name %v: %+v", name, info)

		return nil
	},
}
