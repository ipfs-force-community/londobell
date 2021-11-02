package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheGetCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_get_ops_total",
		Help: "The total number of cache get req",
	})

	CacheGetMissCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_get_miss_ops_total",
		Help: "The total number of cache get missed req",
	})

	CacheViewCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_view_ops_total",
		Help: "The total number of cache view req",
	})

	CacheViewMissCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_view_miss_ops_total",
		Help: "The total number of cache view missed req",
	})

	CacheHasCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_has_ops_total",
		Help: "The total number of cache has req",
	})

	CacheHasMissCnt = promauto.NewCounter(prometheus.CounterOpts{
		Name: "londo_bell_cache_has_miss_ops_total",
		Help: "The total number of cache has missed req",
	})
)
