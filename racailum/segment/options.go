package segment

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

type extractOptions struct {
	Confidence  abi.ChainEpoch
	MinPeriod   abi.ChainEpoch
	MaxBackward abi.ChainEpoch

	TipSetJobLimit int
	StateJobLimit  int

	TipSetPartSizeLimit int

	// 作业级超时与重试。
	// 背景（2026-10-08 主网事故）：单个 tipset 的抽取内部要走节点 RPC，而 go-jsonrpc
	// 在连接异常时会卡在等 nil channel 上永不返回；原实现给作业的 context 只有 cancel
	// 没有 deadline，于是一次卡死 = 整批永久停摆、游标不动、进程却"看着还活着"。
	// 现在每次尝试都有独立 deadline，超时即放弃该次尝试并重试；重试用尽返回错误，
	// 上层（Run 循环）会在下一个 tipset 到来时按游标整段重跑，实现自愈。
	TipSetJobTimeout time.Duration
	StateJobTimeout  time.Duration
	JobRetry         int

	ExtractOptions extract.Options

	OnlyExtractState bool
}

type persistOptions struct {
	Async            bool
	AsyncState       bool
	BatchInsertLimit int

	// 落库等待上限：Mongo 写入挂住时同样不能永久等待（异步等待与同步插入都受它约束）。
	WaitTimeout time.Duration
}

// Options for segment
type Options struct {
	Extract extractOptions

	Persist persistOptions

	Aggregate aggregate.Options

	AllToCheckTableList []string
}

// DefaultOptions returns a default instance of the Options
func DefaultOptions() Options {
	opt := Options{
		Extract: extractOptions{
			Confidence:  10,
			MinPeriod:   20,
			MaxBackward: 50000,

			TipSetJobLimit: 8,
			StateJobLimit:  32,

			TipSetPartSizeLimit: 16,

			// 20 分钟对「日边界全量 actor 余额」那类重 tipset 留了约 3 倍余量
			// （线上实测该 tipset 约 6~8 分钟）；重试 1 次 = 最多 2 次尝试。
			TipSetJobTimeout: 20 * time.Minute,
			StateJobTimeout:  5 * time.Minute,
			JobRetry:         1,

			ExtractOptions:   extract.DefaultOptions(),
			OnlyExtractState: false,
		},

		Persist: persistOptions{
			Async:            true,
			AsyncState:       false,
			BatchInsertLimit: 4 << 10,
			// 60 分钟（原 10 分钟）：线上实测「日边界全量 actor」那类重批的落库尾巴会超过 10 分钟
			// （2026-10-09 主网：2053 高度的大批跑到 128/129 分片时被 10 分钟上限作废，整批重来，
			//  而重试区间的上沿随链头增长 ⇒ 结构上永远追不上，实测 16.6 小时内 32 次全败）。
			// 抽取侧的「作业卡死」仍由 TipSetJobTimeout / StateJobTimeout 兜住，这里放宽的只是落库等待。
			WaitTimeout: 60 * time.Minute,
		},

		AllToCheckTableList: []string{
			"ActorBalance",
			"ActorState",
			"Allocations",
			"ClaimedPower",
			"Claims",
			"DatacapAllowances",
			"DatacapBalances",
			"DealProposal",
			"DealProposalDetail",
			"DealProposalSummary",
			"ExecTrace",
			"FilSupply",
			"MarketFunds",
			"Message",
			"MinerDealSector",
			"MinerFunds",
			"MinerSectorHealth",
			"MinerSectorSummary",
			"MiningProfitability",
			"MultisigBalance",
			"PendingTxns",
			"Tipset",
			"VerifiedRegistry",
		},
	}

	return opt
}

// OptionFn modifies some field of the options
type OptionFn func(*Options)

// TipSetJobLimit override the TipSetJobLimit by the given non-zero limit
func TipSetJobLimit(limit int) OptionFn {
	return func(opt *Options) {
		if limit > 0 {
			opt.Extract.TipSetJobLimit = limit
		}
	}
}

// StateJobLimit override the StateJobLimit by the given non-zero limit
func StateJobLimit(limit int) OptionFn {
	return func(opt *Options) {
		if limit > 0 {
			opt.Extract.StateJobLimit = limit
		}
	}
}

// TipSetJobTimeout override the per-tipset job timeout by the given positive duration
func TipSetJobTimeout(d time.Duration) OptionFn {
	return func(opt *Options) {
		if d > 0 {
			opt.Extract.TipSetJobTimeout = d
		}
	}
}

// StateJobTimeout override the per-regular-state job timeout by the given positive duration
func StateJobTimeout(d time.Duration) OptionFn {
	return func(opt *Options) {
		if d > 0 {
			opt.Extract.StateJobTimeout = d
		}
	}
}

// JobRetry override the per-job retry count (non-negative)
func JobRetry(n int) OptionFn {
	return func(opt *Options) {
		if n >= 0 {
			opt.Extract.JobRetry = n
		}
	}
}

// PersistWaitTimeout override the persist wait timeout by the given positive duration
func PersistWaitTimeout(d time.Duration) OptionFn {
	return func(opt *Options) {
		if d > 0 {
			opt.Persist.WaitTimeout = d
		}
	}
}
