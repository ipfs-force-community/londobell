package tracing

import (
	"net/http"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/trace"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/filecoin-project/lotus/lib/tracing"
)

var log = logging.Logger("ra-tracing")

// just for record current sample rate, not concurrency-safety
// necessary to add lock?
var curRate float64

func Setup(opt *Options, mux *http.ServeMux) *tracesdk.TracerProvider {
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
	curRate = f
	//this should override the setup inside tracing.SetupJaegerTracing
	trace.ApplyConfig(trace.Config{
		DefaultSampler: sampler,
	})

	return
}

func SetSampleRate(f float64) (old float64, err error) {
	old = curRate
	sampler := MixinSample(logicAnd, FilterSample(defaultKeys), trace.ProbabilitySampler(f))
	trace.ApplyConfig(trace.Config{
		DefaultSampler: sampler,
	})
	curRate = f
	return old, err
}

func GetSampleRate() (float64, error) {
	return curRate, nil
}
