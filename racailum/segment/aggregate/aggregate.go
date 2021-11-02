package aggregate

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/limiter"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/jsbson"
)

var log = logging.Logger("aggregate")

const (
	aggregateSubDir = "assets/aggregate"

	aggregateScriptFileExt     = ".js"
	aggregateScriptFileExtLen  = len(aggregateScriptFileExt)
	aggregateScriptFilePattern = "*" + aggregateScriptFileExt
)

// DefaultOptions constructs default options
// script names should be in the format "collection-name.action-name.json"
func DefaultOptions() Options {
	return Options{
		Dir:      "./" + aggregateSubDir,
		JobLimit: 64,
	}
}

// Options for Aggregator
type Options struct {
	Dir      string
	JobLimit int
}

func newCtx(ts *common.LinkedTipSet) Ctx {
	return Ctx{
		Epoch:         int64(ts.Height()),
		MinTimestamp:  ts.MinTimestamp(),
		ParentBaseFee: ts.MinTicketBlock().ParentBaseFee.String(),
	}
}

// Ctx context for Aggregator
type Ctx struct {
	Epoch         int64
	MinTimestamp  uint64
	ParentBaseFee string
}

// Source source content of aggregator script
type Source struct {
	Collection string
	Action     string
	Code       string
}

// New constructs an Aggregator
func New(opt Options, db common.DocumentDB) (*Aggregator, error) {
	return &Aggregator{
		opt:     opt,
		db:      db,
		pattern: filepath.Join(opt.Dir, aggregateScriptFilePattern),
	}, nil
}

// Aggregator is the aggregation executor
type Aggregator struct {
	opt Options
	db  common.DocumentDB

	pattern string
}

// Aggregate tries to execute aggregations based on given tipsets
func (a *Aggregator) Aggregate(ctx context.Context, tss []*common.LinkedTipSet) error {
	if len(tss) == 0 {
		return nil
	}

	alog := log.With("range", common.FormatTipSetEpochRange(tss))

	start := time.Now()

	sources, err := a.loadSources(ctx)
	if err != nil {
		return err
	}

	par := limiter.NewParallel(ctx, a.opt.JobLimit)
	defer par.Finish()

	var done int32
	for ti := range tss {
		ts := tss[ti]
		actx := newCtx(ts)

		for si := range sources {
			src := sources[si]

			par.P(func(ictx context.Context) error {

				pipeline, err := jsbson.Parse(actx, src.Code)
				if err != nil {
					return fmt.Errorf("parse pipeline for %s: %w", src.Action, err)
				}

				err = a.db.Aggregate(ictx, src.Collection, pipeline, nil)
				if err != nil {
					return fmt.Errorf("aggregate: %w", err)
				}

				atomic.AddInt32(&done, 1)

				return nil
			})
		}
	}

	err = par.Wait()

	alog.Infow("all aggregations finished", "elapsed", time.Now().Sub(start).String(), "ts", len(tss), "src", len(sources), "done", done, "has-err", err != nil)
	if err != nil {
		return err
	}

	return nil
}

func (a *Aggregator) loadSources(ctx context.Context) ([]Source, error) {
	matches, err := filepath.Glob(a.pattern)
	if err != nil {
		return nil, fmt.Errorf("search for matched files: %w", err)
	}

	contents := make([]Source, 0, len(matches))
	for _, match := range matches {
		fname := filepath.Base(match)
		pieces := strings.SplitN(fname[:len(fname)-aggregateScriptFileExtLen], ".", 2)
		col := strings.TrimSpace(pieces[0])

		if col == "" {
			log.Warnw("ignore invalid json file", "path", match)
			continue
		}

		var action string
		if len(pieces) > 1 {
			action = pieces[1]
		}

		content, err := ioutil.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("read content of %s: %w", match, err)
		}

		contents = append(contents, Source{
			Collection: col,
			Action:     action,
			Code:       string(content),
		})
	}

	return contents, nil
}
