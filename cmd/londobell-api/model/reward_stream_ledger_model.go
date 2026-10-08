package model

// RewardStreamLedgerRes 是 londobell 端点 POST /adapter/reward_stream_ledger 的响应数据
// （NV29 三流奖励批次 2 · 契约 D §8.1）。
//
// 金额一律 attoFIL 十进制字符串（奖励计数器与余额都超出 int64），权重与份额是
// Denom=1e18 定点的十进制字符串。f02 仍是 v18 actor（NV29 未激活，如当前主网）时
// NV29=false、Streams/Tombstones 为空数组、Liability="0"，不报错。
type RewardStreamLedgerRes struct {
	Epoch      int64                   `json:"epoch"`
	NV29       bool                    `json:"nv29"`
	Denom      string                  `json:"denom"`
	Streams    []RewardLedgerStream    `json:"streams"`
	Tombstones []RewardLedgerTombstone `json:"tombstones"`
	Liability  string                  `json:"liability"`
}

// RewardLedgerStream 是一条活跃奖励流的账本行。隐式流（付给爆块矿工的共识流）没有
// Distribution，因此 Accrued/Payable/ClaimedPeriod 恒为 "0"、Recipients 为空。
type RewardLedgerStream struct {
	ID              int64                   `json:"id"`
	Implicit        bool                    `json:"implicit"`
	EvaluatedWeight string                  `json:"evaluated_weight"`
	Accrued         string                  `json:"accrued"`
	ClaimedPeriod   string                  `json:"claimed_period"`
	Payable         string                  `json:"payable"`
	Recipients      []RewardLedgerRecipient `json:"recipients"`
}

// RewardLedgerRecipient 是显式流的一个受益方：本期份额 + 历史结转待付 + 本期已提。
type RewardLedgerRecipient struct {
	Address       string `json:"address"`
	Share         string `json:"share"`
	Payable       string `json:"payable"`
	ClaimedPeriod string `json:"claimed_period"`
}

// RewardLedgerTombstone 是一条已移除流的遗留负债（受益方仍可提取）。
type RewardLedgerTombstone struct {
	ID         int64                       `json:"id"`
	Recipients []RewardLedgerTombRecipient `json:"recipients"`
}

// RewardLedgerTombRecipient 是 tombstone 的一个受益方及其未提余额。
type RewardLedgerTombRecipient struct {
	Address string `json:"address"`
	Payable string `json:"payable"`
}
