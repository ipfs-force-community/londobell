package model

// RewardStreamPoint 是 /aggregators/reward_streams 的单行小时快照。
//
// 四个计数器一律以字符串返回（余额/奖励均为超出 int64 的大整数），
// 缺失的字段统一填 "0"：调用方按固定结构解码，无需区分 v18/v19。
type RewardStreamPoint struct {
	Epoch                   int64  `json:"Epoch"`
	TotalStoragePowerReward string `json:"TotalStoragePowerReward"` // v18 及以前；v19 给 "0"
	TotalMintedReward       string `json:"TotalMintedReward"`       // v19；v18 给 "0"
	TotalBurnMinted         string `json:"TotalBurnMinted"`         // v19；v18 给 "0"
	TotalExplicitMinted     string `json:"TotalExplicitMinted"`     // v19；v18 给 "0"
}
