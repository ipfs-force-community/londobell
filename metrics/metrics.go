package metrics

import (
	"context"

	"github.com/filecoin-project/lotus/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	Timer               = metrics.Timer
	SinceInMilliseconds = metrics.SinceInMilliseconds
)

var (
	LoadTipSetDuration  = stats.Float64("load_tipset_duration_ms", "Duration of lotus load tipset request", stats.UnitMilliseconds)
	TipSetHeight        = stats.Int64("tipset_height", "The lastest received tipset's height ", stats.UnitDimensionless)
	ExtractDuration     = stats.Float64("persist_duration_ms", "Duration of a models persist operation", stats.UnitMilliseconds)
	ExtractError        = stats.Int64("extract_error_status", "Status of extract tipset", stats.UnitDimensionless)
	LowerBoundary       = stats.Int64("lower_boundary", "lower boundary of segment", stats.UnitDimensionless)
	UpperBoundary       = stats.Int64("upper_boundary", "upper boundary of segment", stats.UnitDimensionless)
	OutdatedFinalHeight = stats.Int64("outdated_final_height", "outdated finalHeight of extract", stats.UnitDimensionless)

	CacheGetCnt      = stats.Int64("cache_get_ops_total", "Cache get count", stats.UnitDimensionless)
	CacheGetMissCnt  = stats.Int64("cache_get_miss_ops_total", "Cache get miss count", stats.UnitDimensionless)
	CacheViewCnt     = stats.Int64("cache_view_ops_total", "Cache view count", stats.UnitDimensionless)
	CacheViewMissCnt = stats.Int64("cache_view_miss_ops_total", "Cache view miss count", stats.UnitDimensionless)
	CacheHasCnt      = stats.Int64("cache_has_ops_count", "Cache has count", stats.UnitDimensionless)
	CacheHasMissCnt  = stats.Int64("cache_has_miss_opts_count", "Cache has miss count", stats.UnitDimensionless)
)
var (
	Cache, _ = tag.NewKey("cache")
)

var DefaultViews = []*view.View{
	{
		Name:        TipSetHeight.Name(),
		Measure:     TipSetHeight,
		Aggregation: view.LastValue(),
	},
	{
		Name:        LoadTipSetDuration.Name(),
		Measure:     LoadTipSetDuration,
		Aggregation: view.Distribution(1, 2, 4, 8, 16, 32, 64),
	},
	{
		Name:        ExtractDuration.Name(),
		Measure:     ExtractDuration,
		Aggregation: view.Distribution(10000, 15000, 20000, 25000, 30000, 40000, 50000, 60000),
	},
	{
		Name:        ExtractError.Name(),
		Measure:     ExtractError,
		Aggregation: view.LastValue(),
	},
	{
		Name:        LowerBoundary.Name(),
		Measure:     LowerBoundary,
		Aggregation: view.LastValue(),
	},
	{
		Name:        UpperBoundary.Name(),
		Measure:     UpperBoundary,
		Aggregation: view.LastValue(),
	},
	{
		Name:        OutdatedFinalHeight.Name(),
		Measure:     OutdatedFinalHeight,
		Aggregation: view.LastValue(),
	},
}

var CacheView = []*view.View{
	{
		Measure:     CacheGetCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
	{
		Measure:     CacheGetMissCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
	{
		Measure:     CacheViewCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
	{
		Measure:     CacheViewMissCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
	{
		Measure:     CacheHasCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
	{
		Measure:     CacheHasMissCnt,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Cache},
	},
}

func RecordInc(ctx context.Context, m *stats.Int64Measure) {
	stats.Record(ctx, m.M(1))
}

func RecordDec(ctx context.Context, m *stats.Int64Measure) {
	stats.Record(ctx, m.M(-1))
}
