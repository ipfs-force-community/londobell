package main

import (
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
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
	l.ZapEventLogger.Debugf(msg, args...)
}

func parsetTipSetKey(s string) (types.TipSetKey, error) {
	cids, err := lcli.ParseTipSetString(s)
	if err != nil {
		return types.EmptyTSK, err
	}

	return types.NewTipSetKey(cids...), nil
}
