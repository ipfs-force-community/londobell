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

	// TODO: register sampler set handler
	mux.HandleFunc("/_tracing", func(rw http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		if _, err := rw.Write([]byte("we should register some http handler for tracing adjustment, but none now")); err != nil {
			log.Warnf("write response body: %s", err)
		}
		return
	})

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
		sampler = trace.AlwaysSample()

	default:
		sampler = trace.ProbabilitySampler(f)
	}

	// this should override the setup inside tracing.SetupJaegerTracing
	trace.ApplyConfig(trace.Config{
		DefaultSampler: sampler,
	})

	return
}
