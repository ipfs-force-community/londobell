package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	cliutil "github.com/filecoin-project/lotus/cli/util"
)

var (
	log   = logging.Logger("bell")
	fxlog = &fxLogger{
		ZapEventLogger: log,
	}
)

type fxLogger struct {
	*logging.ZapEventLogger
}

// Printf impls fx.Printer.Printf
func (l *fxLogger) Printf(msg string, args ...interface{}) {
	l.ZapEventLogger.Infof(msg, args...)
}

func parsetTipSetKey(s string) (types.TipSetKey, error) {
	cids, err := lcli.ParseTipSetString(s)
	if err != nil {
		return types.EmptyTSK, err
	}

	return types.NewTipSetKey(cids...), nil
}

func getRepoHomeDir(cctx *cli.Context) (string, error) {
	dir, err := homedir.Expand(cctx.String(repoFlag.Name))
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

func getFullNode(cctx *cli.Context) (api.FullNode, jsonrpc.ClientCloser, error) {
	return cliutil.GetFullNodeAPI(cctx)
}
