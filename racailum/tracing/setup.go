package tracing

import (
	"net/http"

	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/filecoin-project/lotus/lib/tracing"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/trace"
)

var log = logging.Logger("ra-tracing")

func Setup(opt *Options, mux *http.ServeMux) *jaeger.Exporter {
	if !opt.Enable || opt.Name == "" {
		return nil
	}

	log.Infow("try to enable jaeger exporter", "name", opt.Name)

	applyEnvOpts(opt)
	exporter := tracing.SetupJaegerTracing(opt.Name)
	applySampler(opt)

	return exporter
}

func applySampler(opt *Options) {
	if opt.Sampler == nil {
		return
	}

	var sampler trace.Sampler
	f := *opt.Sampler
	log.Infof("set trace sampler probability", "value", f)

	switch {
	case f <= 0:
		sampler = trace.NeverSample()

	case f >= 1:
		sampler = FilterSample(defaultKeys)

	default:
		sampler = MixinSample(logicAnd, sampler, trace.ProbabilitySampler(f))
	}

	//this should override the setup inside tracing.SetupJaegerTracing
	trace.ApplyConfig(trace.Config{
		DefaultSampler: sampler,
	})

	return
}
