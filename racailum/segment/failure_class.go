package segment

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"
)

// 失败类别常量。只有 state / data 属于「可登记类别」（= 允许登记台账并继续往后抽）。
const (
	FailureClassState     = "state"
	FailureClassData      = "data"
	FailureClassTransport = "transport"
	FailureClassUnknown   = "unknown"
)

// 阻断面来源（PartFailure.Kind 的取值）。
//
// 阻断面 = 本轮「不能越过、下一轮必须重做的起始高度」，其来源 = 已登记的洞 ∪ 失败 part。
// 二者都进阻断清单，但处置不同：洞写台账（等 heal 闭环销项），失败 part 不写台账（下一轮重做）。
const (
	// BlockedByHole：可登记类别且已写入台账的洞（既有语义不变，仍由 heal 闭环销项）。
	BlockedByHole = "hole"
	// BlockedByPersistFailure：不可登记 / 未开启的 part 失败（含传输类、超时、未知）。
	// 不写台账——它是「下一轮要重做」，不是永久的洞；只作为本轮发布的截断点。
	BlockedByPersistFailure = "persist-failure"
)

// maxErrDigestLen 是错误摘要的保留上限。台账是给人看/给机器归因用的，
// 不需要（也不应该）把可能长达数百 KB 的 mongo 批量写错误原文存进去。
const maxErrDigestLen = 500

// PartFailure 描述一次「part 抽取失败」。写入台账的字段就是它的子集。
type PartFailure struct {
	FromEpoch abi.ChainEpoch // part 起始高度
	ToEpoch   abi.ChainEpoch // part 结束高度
	Class     string         // "state" | "data" | "transport" | "unknown"
	ErrDigest string         // 错误摘要（截断，勿存超长）
	CID       string         // 从错误文本里提的根 CID，提不到则空
	Attempts  int            // 观察到的尝试次数（分类时为 1，台账里累加）

	// Kind 标注该失败作为「阻断面」时的来源：BlockedByHole（已登记台账的洞）或
	// BlockedByPersistFailure（未登记的失败 part，下一轮重做）。它不是写入台账的字段子集
	// （台账只认可登记的洞）。零值按 BlockedByHole 处理，使既有只填 Class/区间按字面量
	// 构造的 PartFailure 语义不变。
	Kind string
}

// classifyPartFailure 对一次 part 失败做分类。
//
// 第二个返回值 = 是否「属可登记类别」（仅 state / data 可能为 true）。
// 这是整套改动的安全阀：**默认为不可登记**——宁可停，不可静默跳。
// 传输类错误（节点 RPC 卡死/超时、mongo 落库超时、连接中断）一律不可登记，
// 因为它们与链上数据无关，登记（= 跳过）会把瞬时故障固化成永久洞。
func classifyPartFailure(part []*common.LinkedTipSet, err error) (PartFailure, bool) {
	pf := PartFailure{Attempts: 1}
	if len(part) > 0 {
		pf.FromEpoch = part[0].Height()
		pf.ToEpoch = part[len(part)-1].Height()
	}

	if err == nil {
		pf.Class = FailureClassUnknown
		return pf, false
	}

	msg := err.Error()
	pf.ErrDigest = truncateErrDigest(msg)
	pf.CID = extractRootCID(msg)

	// 1) 作业超时必须用类型判定（errors.As），不能靠字符串：
	//    它是「节点 RPC 卡死/超时」这一传输类故障，与数据本身无关。
	var te *extractJobTimeoutErr
	if errors.As(err, &te) {
		pf.Class = FailureClassTransport
		return pf, false
	}

	switch {
	// 2) state：状态树/HAMT 节点缺失 —— 典型形态是 cali 那类缺块（本次改造要处理的正是它）。
	case containsAny(msg, "ipld: could not find", "failed to load hamt node", "failed to load state tree"):
		pf.Class = FailureClassState
		return pf, true

	// 3) data：数据本身解码失败 —— 与 state 同类，可登记。
	case containsAny(msg, "not minimally encoded", "failed to unmarshal"):
		pf.Class = FailureClassData
		return pf, true

	// 4) transport：落库超时 / 上下文超时 / 连接中断 —— 不可登记。
	case containsAny(msg, "async persist wait timeout", "context deadline exceeded") ||
		(strings.Contains(msg, "connection") && strings.Contains(msg, "incomplete read")):
		pf.Class = FailureClassTransport
		return pf, false

	// 5) 未知：保守当作不可登记（默认整批弃）。
	default:
		pf.Class = FailureClassUnknown
		return pf, false
	}
}

// ClassIsRecordable 报告某个类别是否属于「可登记类别」。与 classifyPartFailure 的第二
// 返回值同源，供调用方 / 测试显式复核「默认不可登记」这一安全属性。
func ClassIsRecordable(class string) bool {
	return class == FailureClassState || class == FailureClassData
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// truncateErrDigest 截断错误摘要到 maxErrDigestLen 字节，并在 UTF-8 边界上收尾，
// 避免把多字节字符劈成半个（mongo 会原样存字节，劈开会污染日志正文）。
func truncateErrDigest(msg string) string {
	if len(msg) <= maxErrDigestLen {
		return msg
	}
	cut := maxErrDigestLen
	for cut > 0 && !utf8.RuneStart(msg[cut]) {
		cut--
	}
	return msg[:cut] + "...<truncated>"
}

// cidCandidateRe 抓错误正文里「像 CID 的 token」：base32(bafy…/bafk…) 或 base58btc(Qm…)。
// 长度下限 20 直接滤掉绝大多数无关短词。
var cidCandidateRe = regexp.MustCompile(`[a-zA-Z0-9]{20,}`)

// extractRootCID 从错误正文里提取第一个能真正解码的 CID；提不到返回空串。
// 用 cid.Decode 复核（而不是纯正则），确保不会把普通长字符串误当 CID。
func extractRootCID(msg string) string {
	for _, cand := range cidCandidateRe.FindAllString(msg, -1) {
		if _, err := cid.Decode(cand); err == nil {
			return cand
		}
	}
	return ""
}

// summarizeFailures 把已登记的失败压成一行摘要，供错误信息与日志使用。
func summarizeFailures(failures []PartFailure) string {
	parts := make([]string, 0, len(failures))
	for _, f := range failures {
		parts = append(parts, fmt.Sprintf("[%d-%d:%s:%s]", f.FromEpoch, f.ToEpoch, f.Class, f.CID))
	}
	return strings.Join(parts, ",")
}
