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

	// 失败分级（§8.2 / §8.7）：允许「登记台账并继续往后抽」的错误类别。
	// 默认必须为空 slice —— 空 = 完全维持现状（不登记、不跳过、失败即整批作废），
	// 这样灰度回退是改配置而不是回滚二进制；确认链路无误后再按 state → data 逐类开。
	// 注意：即便某类在此开启，仍需 classifyPartFailure 判定它属「可登记类别」才生效。
	SkippableErrorClasses []string

	// 护栏（§8.5）：未发布区间（链头 − 已发布水位）上限，超过即停止继续往后抽并告警。
	// 默认 2000 高度（约 17 小时）。<=0 视为关闭护栏。
	MaxUnpublishedRange abi.ChainEpoch

	// 轮间退避（§8.6）：某轮失败后不再「下个 tipset（约 30 秒）到来即重跑整段」，
	// 而是按指数退避等待，成功一轮立即复位。理由：失败轮都在已知会失败的高度上白跑
	// ——实测 cali 每轮白跑 826 个高度、并持续消耗 2 核 mongo 的算力。退避不减少
	// 失败次数，但能把无效消耗降到接近 0。
	// 这两个字段只是为了把退避曲线可配置化，默认值与改动前的「无等待」相比更保守
	// （更保守 = 失败时少空转），且不含任何「跳过/degraded」语义。
	RoundBackoffInitial time.Duration // 默认 5m
	RoundBackoffMax     time.Duration // 默认 30m
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

			// 60 分钟：给「整点/日边界全量 actor 快照」那类重 tipset 留足余量（实测 30~40 分钟
			// 的 1.5~2 倍）。2026-10-20 NV29 激活后 v19 状态树更大，且与 actor 同步器的状态差异
			// 计算争抢 lotus 节点，该 tipset 单 job 曾达 30~40 分钟；旧的 20 分钟阈值把正常长任务
			// 判成 transport 失败 ⇒ 该轮被标为 blocking boundary、提交水位钉死在整点前一个高度，
			// 下游 miner/pro/sector/erc20/fns/nft 全部停摆（只能重启 bell 清内存态标记恢复）。
			// 超时只在 job 真挂死时才起作用（最多 2 次尝试），宁可等也不必误杀。重试次数不变。
			TipSetJobTimeout: 60 * time.Minute,
			StateJobTimeout:  5 * time.Minute,
			JobRetry:         1,

			ExtractOptions:   extract.DefaultOptions(),
			OnlyExtractState: false,

			// 失败分级默认关闭：空 slice = 完全维持现状（不登记、不跳过）。
			SkippableErrorClasses: nil,
			// 护栏：未发布区间上限 2000 高度（约 17 小时）。
			MaxUnpublishedRange: 2000,

			// 退避曲线：5m → 10m → 20m → 上限 30m（见 §8.6）。取值的根据是
			// 「配置默认值必须与现状等价或更保守」——失败轮每轮都会在已知失败的高度上
			// 白跑数十分钟，退避正是用来砍掉这段无效消耗，故上限取到 30m 而非更短。
			RoundBackoffInitial: 5 * time.Minute,
			RoundBackoffMax:     30 * time.Minute,
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
			"ActorBalanceTop",
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

// RoundBackoffInitial override the inter-round backoff initial duration by the given positive duration
func RoundBackoffInitial(d time.Duration) OptionFn {
	return func(opt *Options) {
		if d > 0 {
			opt.Extract.RoundBackoffInitial = d
		}
	}
}

// RoundBackoffMax override the inter-round backoff upper bound by the given positive duration
func RoundBackoffMax(d time.Duration) OptionFn {
	return func(opt *Options) {
		if d > 0 {
			opt.Extract.RoundBackoffMax = d
		}
	}
}

// SkippableErrorClasses override the error classes that are allowed to be recorded and skipped.
// 空列表 = 完全维持现状（不登记、不跳过）——这也是默认值，灰度回退即改回空。
// 传入 nil/空即显式关闭（覆盖默认值不改语义，因为默认本就是空）。
func SkippableErrorClasses(classes ...string) OptionFn {
	return func(opt *Options) {
		opt.Extract.SkippableErrorClasses = append([]string(nil), classes...)
	}
}

// MaxUnpublishedRange override the guard threshold (未发布区间上限) by the given non-negative value.
// 0 视为关闭护栏（见 extract.go 的 partRunner.guardExceeded）。
func MaxUnpublishedRange(r abi.ChainEpoch) OptionFn {
	return func(opt *Options) {
		if r >= 0 {
			opt.Extract.MaxUnpublishedRange = r
		}
	}
}
