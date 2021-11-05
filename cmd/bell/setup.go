package main

import (
	"net/http"
	"net/http/pprof"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/ipfs-force-community/londobell/metrics"
	"go.opencensus.io/stats/view"
)

func setupMetrics(opts metrics.Options) error {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "londobell",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	// register the metrics views of interest
	var views []*view.View
	views = append(views, metrics.DefaultViews...)
	views = append(views, metrics.CacheView...)
	if err := view.Register(views...); err != nil {
		log.Fatalf("Failed to register views: %v", err)
		return err
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		mux.HandleFunc("/debug/pprof", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		if err := http.ListenAndServe(opts.PrometheusPort, mux); err != nil {
			log.Fatalf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()
	return nil
}
