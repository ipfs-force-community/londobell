package tracing

import (
	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/filecoin-project/lotus/lib/tracing"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("ra-tracing")

func Setup(opt *Options) *jaeger.Exporter {
	if !opt.Enable || opt.Name == "" {
		return nil
	}

	log.Infow("try to enable jaeger exporter", "name", opt.Name)

	applyEnvOpts(opt)
	exporter := tracing.SetupJaegerTracing(opt.Name)
	return exporter
}
