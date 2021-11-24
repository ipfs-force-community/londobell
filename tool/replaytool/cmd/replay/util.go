package main

import (
	logging "github.com/ipfs/go-log/v2"
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
	l.ZapEventLogger.Debugf(msg, args...)
}
