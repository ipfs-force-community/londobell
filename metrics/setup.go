package metrics

import (
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats/view"
)

var log = logging.Logger("ra-metrics")

const defaultNamespace = "londobell"

type Options struct {
	Enable    bool
	Namespace string
}

func DefaultOptions() Options {
	return Options{
		Enable:    true,
		Namespace: defaultNamespace,
	}
}

func Setup(opt *Options, mux *http.ServeMux) error {
	if !opt.Enable || opt.Namespace == "" {
		return nil
	}

	log.Infow("try to enable prometheus exporter", "ns", opt.Namespace)

	exporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: opt.Namespace,
	})

	if err != nil {
		return fmt.Errorf("construct exporter: %w", err)
	}

	var views []*view.View
	views = append(views, DefaultViews...)
	views = append(views, CacheView...)

	if err := view.Register(views...); err != nil {
		return fmt.Errorf("register views: %w", err)
	}

	mux.Handle("/_metrics", exporter)

	return nil
}
