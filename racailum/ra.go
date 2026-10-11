package racailum

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum/grafana"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/tracing"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	lconfig "github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

var log = logging.Logger("racailum")

const DefaultHTTPListenAddr = ":15002"
const DefaultRPCListenAddr = "/ip4/127.0.0.1/tcp/12345"

// nextBackoff 计算「连续失败 consecutive 次后」该等待的轮间退避时长（纯函数）。
//
// 为什么抽成纯函数：退避是「不睡眠就无法观察」的时序逻辑，抽出来后可脱离时钟、
// 用表驱动单测锁住曲线与封顶，避免为了测一条曲线去起整个 Run 循环。
//
// 曲线：consecutive 从 1 开始 ⇒ 1 → initial，之后每多一次失败翻倍，直到 max 封顶。
//
//	nextBackoff(1, 5m, 30m) = 5m
//	nextBackoff(2, 5m, 30m) = 10m
//	nextBackoff(3, 5m, 30m) = 20m
//	nextBackoff(4, 5m, 30m) = 30m
//	nextBackoff(n>=4, 5m, 30m) = 30m
//
// 边界：consecutive < 1 视为 1（第一次失败）；initial/max 非正视为配置异常，
// 退化为「不退避」（返回 0），与改动前的行为等价——绝不因配置把循环卡死。
func nextBackoff(consecutive int, initial, max time.Duration) time.Duration {
	if initial <= 0 || max <= 0 {
		return 0
	}
	if consecutive < 1 {
		consecutive = 1
	}

	d := initial
	for i := 1; i < consecutive; i++ {
		if d >= max {
			break
		}
		d *= 2
	}
	if d > max {
		d = max
	}
	return d
}

// roundBackoffState 是 Run 循环持有的轮间退避状态：失败递增、成功复位。
// 抽成独立对象是为了让「成功一轮立即复位」这条调用方逻辑也有自动化覆盖
// ——否则只能起整个 Run 循环（依赖 cs/tsCh）才能验证复位。
type roundBackoffState struct {
	initial     time.Duration
	max         time.Duration
	consecutive int
}

// fail 记一次失败并返回本次该退避的时长（由 nextBackoff 按曲线算出）。
func (b *roundBackoffState) fail() time.Duration {
	b.consecutive++
	return nextBackoff(b.consecutive, b.initial, b.max)
}

// success 记一次成功：连续失败计数归零，下一次失败重新从 initial 起算。
func (b *roundBackoffState) success() {
	b.consecutive = 0
}

// SegmentConfig contains info about a segment
type SegmentConfig struct {
	DSN     string
	ReadDSN string
	Name    string
}

// DefaultConfig returns the default config
func DefaultConfig() Config {
	return Config{
		HTTP:           DefaultHTTPOptions(),
		Grafana:        grafana.DefaultOptions(),
		Aggregate:      aggregate.DefaultOptions(),
		Segment:        segment.DefaultOptions(),
		Metrics:        metrics.DefaultOptions(),
		Tracing:        tracing.DefaultOptions(),
		EnableTracing:  true,
		EnableGrafana:  true,
		EnableDebug:    true,
		OutdatedGap:    240,
		TriggerSpan:    240, // 2h
		TempDBCapacity: 240,
	}
}

type HTTPOptions struct {
	RPCListen  string
	Listen     string
	StableWait lconfig.Duration
}

func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		RPCListen:  DefaultRPCListenAddr,
		Listen:     DefaultHTTPListenAddr,
		StableWait: lconfig.Duration(5 * time.Second),
	}
}

// Config of RaCailum
type Config struct {
	HTTP           HTTPOptions
	Grafana        grafana.Options
	Aggregate      aggregate.Options
	Segment        segment.Options
	Metrics        metrics.Options
	Tracing        tracing.Options
	EnableTracing  bool
	EnableGrafana  bool
	EnableDebug    bool
	OutdatedGap    int64
	TriggerSpan    uint
	TempDBCapacity uint
}

// New returns an instance of *RaCailum
func New(ctx context.Context, cfg Config, sub common.HeadNotifier, cs common.ChainStore, stm common.StateManager, segmgr *segment.Manager, shutdownCh dtypes.ShutdownChan, full common.FullNodeApiGetter) (*RaCailum, error) {
	vm.EnableDetailedTracing = cfg.EnableTracing

	activeSegName, has, err := segmgr.LoadActive()
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, fmt.Errorf("no active segment")
	}

	activeSeg, err := segment.New(ctx, activeSegName, cfg.Segment, cfg.Aggregate, segmgr, cs, stm, full)
	if err != nil {
		return nil, err
	}

	// gr, err := grafana.New(ctx, cfg.Grafana, []*segment.Segment{activeSeg})
	// if err != nil {
	//     return nil, fmt.Errorf("construct garfana: %w", err)
	// }

	// if cfg.EnableGrafana {
	//     go func() {
	//         if err := gr.Run(ctx); err != nil {
	//             log.Errorf("grafana: %s", err)
	//         }
	//     }()
	// }
	log.Infow("ra sets sail", "tracing", cfg.EnableTracing, "active-seg", activeSegName)
	ra := &RaCailum{
		cfg:        cfg,
		sub:        sub,
		activeSeg:  activeSeg,
		shutdownCh: shutdownCh,
	}

	ra.components.cs = cs
	ra.components.stm = stm
	// ra.gr = gr

	return ra, nil
}

// RaCailum manages the segments
type RaCailum struct {
	cfg Config

	sub common.HeadNotifier

	components struct {
		cs     common.ChainStore
		stm    common.StateManager
		optfns []segment.OptionFn
	}

	// gr        *grafana.Grafana
	activeSeg *segment.Segment

	shutdownCh dtypes.ShutdownChan
}

// Extract is used to extract from given tipset manually
func (r *RaCailum) Extract(ctx context.Context, ts *types.TipSet) error {
	return r.activeSeg.Extract(ctx, ts)
}

// OfflineExtract is used to extract from given tipset key (hi, from)
func (r *RaCailum) WalkExtract(ctx context.Context, from types.TipSetKey, hi abi.ChainEpoch) error {
	ts, err := r.components.cs.LoadTipSet(ctx, from)
	if err != nil {
		return fmt.Errorf("load ts failed, is it exist in this db? : %w", err)
	}
	tipsets, err := segment.ExtractLinkedTipSets(r.components.cs, ts, &hi, false)
	if err != nil {
		return fmt.Errorf("extract linked tipsets failed: %w", err)
	}

	for i := 1; i < len(tipsets); i++ {
		tipsets[i].Parent = tipsets[i-1].TipSet
	}

	tipsets = tipsets[1:]

	if _, err := r.activeSeg.ExtractTipSets(ctx, tipsets, false); err != nil {
		return err
	}

	if err := r.activeSeg.Aggregate(ctx, tipsets); err != nil {
		return err
	}
	return nil
}

// Aggregate is used to trigger aggregations from given tipset boundaries manually
func (r *RaCailum) Aggregate(ctx context.Context, lo, hi *types.TipSet) error {
	loEpoch := lo.Height()
	tss, err := segment.ExtractLinkedTipSets(r.components.cs, hi, &loEpoch, false)
	if err != nil {
		return fmt.Errorf("load tipsets: %w", err)
	}

	return r.activeSeg.Aggregate(ctx, tss)
}

// DryState runs a dry extraction from given ts
func (r *RaCailum) DryState(ctx context.Context, ts *common.LinkedTipSet, allowNilChild bool) ([]*extract.Res, error) {
	return r.activeSeg.DryExtract(ctx, ts, allowNilChild)
}

func (r *RaCailum) Run(ctx context.Context, doneCh <-chan struct{}, tsCh <-chan types.TipSetKey) {
	// 轮间退避状态：跨 tipset 事件保留，失败递增、成功复位。
	backoff := roundBackoffState{
		initial: r.cfg.Segment.Extract.RoundBackoffInitial,
		max:     r.cfg.Segment.Extract.RoundBackoffMax,
	}
HEAD_LOOP:
	for {
		select {
		case <-doneCh:
			log.Info("quit head-change loop")
			return

		case tsk, ok := <-tsCh:
			if !ok {
				log.Warn("tsk chan closed")
				return
			}
			lstart := time.Now()
			ts, err := r.components.cs.LoadTipSet(ctx, tsk)
			stats.Record(ctx, metrics.LoadTipSetDuration.M(metrics.SinceInMilliseconds(lstart)))
			if err != nil {
				log.Errorf("failed to load tipset %s: %s", tsk, err)
				continue HEAD_LOOP
			}

			log.Infow("incoming tipset", "tsk", tsk, "height", ts.Height())
			estart := time.Now()
			if err := r.Extract(ctx, ts); err != nil {
				log.Errorf("failed to persist tipset: %s", err)
				stats.Record(ctx, metrics.ExtractError.M(1))

				// 失败一轮后按指数退避等待（§8.6），而不是「下个 tipset（约 30 秒）到来即重跑整段」：
				// 失败轮都在已知会失败的高度上白跑（实测 cali 每轮白跑 826 个高度并持续消耗 mongo 算力），
				// 退避不减少失败次数，但能把无效消耗降到接近 0。
				// 注意：退避只影响「抽下一轮」，不动提交语义、不改 FinalHeight —— 水位仍停在洞前。
				if d := backoff.fail(); d > 0 {
					log.Warnw("extract round failed, backing off before next round",
						"consecutive-failures", backoff.consecutive, "backoff", d.String())
					// 退避期间仍可被 ctx 取消或 doneCh 关闭打断，避免关不干净。
					// 退避期间 AlertOutdatedFinalHeight 独立在跑，告警不会哑掉。
					select {
					case <-time.After(d):
					case <-ctx.Done():
						log.Info("quit head-change loop during backoff")
						return
					case <-doneCh:
						log.Info("quit head-change loop during backoff")
						return
					}
				}
			} else {
				backoff.success()
				stats.Record(ctx, metrics.ExtractError.M(0))
				stats.Record(ctx, metrics.TipSetHeight.M(int64(ts.Height())))
				stats.Record(ctx, metrics.ExtractDuration.M(metrics.SinceInMilliseconds(estart)))
			}
			log.Infow("done tipset extracting", "tsk", tsk, "height", ts.Height(), "elapsed", time.Now().Sub(estart).String())
		}
	}
}

func (r *RaCailum) AlertOutdatedFinalHeight(ctx context.Context, outdatedGap int64) {
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			var (
				finalHeight abi.ChainEpoch
				err         error
			)

			if r.cfg.Segment.Extract.OnlyExtractState {
				if r.cfg.Segment.Extract.ExtractOptions.EnabelExtract.EnableExtractState {
					finalHeight, err = r.activeSeg.GetStateFinalHeight(ctx)
					if err != nil {
						log.Errorf("get state final height failed: %v", err)
						stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
						continue
					}
				}
			} else {
				finalHeight, err = r.activeSeg.GetFinalHeight(ctx)
				if err != nil {
					log.Errorf("get final height failed: %v", err)
					stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
					continue
				}
			}

			curEpoch := common.GetCurEpoch()
			if curEpoch-finalHeight >= abi.ChainEpoch(outdatedGap) {
				log.Warnf("finalHeight %v lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
				stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
				continue
			}

			log.Infof("finalHeight %v not lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
			stats.Record(ctx, metrics.OutdatedFinalHeight.M(0))
		case <-ctx.Done():
			return
		}
	}
}

// Grafana returns the instance of *Grafana
// func (r *RaCailum) Grafana() *grafana.Grafana {
//     return r.gr
// }
