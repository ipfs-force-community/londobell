package main

import (
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var (
	log   = logging.Logger("replay")
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

func getFullNode(cctx *cli.Context) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	return cliutil.GetFullNodeAPI(cctx)
}
