package grafana

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/dtynn/londobell/lib/limiter"
	"github.com/dtynn/londobell/lib/mgoutil/jsbson"
	"github.com/dtynn/londobell/racailum/segment"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("grafana")

const (
	scriptSubDir = "assets/grafana"
)

// DefaultOptions return the defaults
func DefaultOptions() Options {
	return Options{
		ListenAddr: "0.0.0.0:15502",
		ScriptDir:  "./" + scriptSubDir,
	}
}

// Options for Grafana
type Options struct {
	ListenAddr string
	ScriptDir  string
}

// New is the constructor a *Grafana
func New(ctx context.Context, opts Options, segs []*segment.Segment) (*Grafana, error) {
	loader, err := newScriptLoader(opts.ScriptDir)
	if err != nil {
		return nil, err
	}

	scripts, err := loader.loadAll(ctx)
	if err != nil {
		return nil, err
	}

	scriptMap := map[string]script{}
	scriptNames := make([]string, 0, len(scripts))

	for si := range scripts {
		s := scripts[si]
		sname := fmt.Sprintf("%s-%s", s.collection, s.action)
		if _, has := scriptMap[sname]; has {
			return nil, fmt.Errorf("dupliacte script named %s", sname)
		}

		scriptMap[sname] = s
		scriptNames = append(scriptNames, sname)
	}

	sort.Strings(scriptNames)
	log.Infof("%d scripts loaded", len(scriptNames))

	return &Grafana{
		opts:    opts,
		loader:  loader,
		scripts: scriptMap,
		snames:  scriptNames,
		segs:    segs,
	}, nil
}

// Grafana is a simple json data source implementation
type Grafana struct {
	opts    Options
	loader  *scriptLoader
	scripts map[string]script
	snames  []string

	segs []*segment.Segment
}

// Run starts the server
func (g *Grafana) Run(ctx context.Context) error {
	log.Info("grafana http server starts")
	defer log.Info("grafana http server stops")

	srv := g.HTTPServer(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		srv.Shutdown(shutdownCtx)
	}()

	return srv.ListenAndServe()
}

// HTTPServer construct a http server
func (g *Grafana) HTTPServer(ctx context.Context) *http.Server {
	hdls := map[string]http.Handler{}
	hdls["/search"] = mustHandler(g.serveSearch)
	hdls["/query"] = mustHandler(g.serveQuery)
	// hdls["/annotations"] = mustHandler(g.serveSearch)
	// hdls["/tag-keys"] = mustHandler(g.serveSearch)
	// hdls["/tag-values"] = mustHandler(g.serveSearch)

	inner := func(rw http.ResponseWriter, r *http.Request) {
		h, ok := hdls[r.URL.Path]
		if !ok {
			hlog.Warnf("handler not found for %s", r.URL.Path)
			return
		}

		h.ServeHTTP(rw, r)
	}

	srv := &http.Server{
		Addr:    g.opts.ListenAddr,
		Handler: safe(cors(http.HandlerFunc(inner))),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	return srv
}

func (g *Grafana) serveSearch(req *searchReq) searchResp {
	if req.Target != "" {
		hlog.Warnf("unexpected target %s", req.Target)
		return nil
	}

	return g.snames
}

func (g *Grafana) serveQuery(ctx context.Context, req *queryReq) queryResp {
	fromEpoch := time2epoch(req.Range.From)
	toEpoch := time2epoch(req.Range.To)

	availableSets := make([]int, 0, len(g.segs))
	for di := range g.segs {
		bound := g.segs[di].ReadBoundary()
		if fromEpoch >= bound.Lo.Epoch || toEpoch <= bound.Lo.Epoch {
			availableSets = append(availableSets, di)
		}
	}

	sets := make([][]pointset, len(req.Targets))

	hlog.Infow("query", "from", fromEpoch, "to", toEpoch, "sets", len(availableSets), "targets", len(req.Targets))

	p := limiter.NewParallel(ctx, 8)
	defer p.Finish()

	for ti := range req.Targets {
		ti := ti
		target := req.Targets[ti]
		act, ok := g.scripts[target.Target]
		if !ok {
			continue
		}

		qctx := queryCtx{
			From: int64(fromEpoch),
			To:   int64(toEpoch),
			Data: target.Data,
		}

		alog := log.With("col", act.collection, "act", act.action)
		alog.Infow("query for points")

		p.P(func(ctx context.Context) error {
			pipeline, err := jsbson.Parse(qctx, act.code)
			if err != nil {
				alog.Errorf("parse pipeline: %s", err)
				return nil
			}

			var res []pointset

			defer func() {
				sets[ti] = res
			}()

			for di := range availableSets {
				var psets []pointset
				if err := g.segs[di].ReadDB().Aggregate(ctx, act.collection, pipeline, &psets); err != nil {
					alog.Errorf("get data from dataset %s: %s", g.segs[di].Name(), err)
					return nil
				}

				res = append(res, psets...)
			}

			alog.Infow("aggregated", "point-sets", len(res))

			return nil
		})
	}

	err := p.Wait()
	maybeAbort(err)

	results := queryResp{}
	for si := range sets {
		if len(sets[si]) > 0 {
			results = append(results, sets[si]...)
		}
	}

	return results
}
