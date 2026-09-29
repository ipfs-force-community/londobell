package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type SectorExpirationRes struct {
	SectorExpirations []SectorOnChainInfo
}

type SectorInfoRes struct {
	SectorExpirationRes
	QAPowerRes
}

type SectorOnChainInfo struct {
	Expiration         abi.ChainEpoch
	Activation         abi.ChainEpoch
	DealWeight         abi.DealWeight
	VerifiedDealWeight abi.DealWeight
	InitialPledge      abi.TokenAmount
	// Flags / PowerBaseEpoch / FullQaPower 是 NV29（Solstice / FIP-0118）之后新增的加性字段：
	// 老字段（Expiration/Activation/DealWeight/VerifiedDealWeight/InitialPledge）语义不变，
	// filscan 侧按同名 JSON 字段解码（filscan_backend/pkg/londobell/adapter.go MinerSector）。
	//
	// Flags = SectorOnChainInfo.Flags（1<<0 SIMPLE_QA_POWER，1<<1 FULL_QA_POWER）；
	// PowerBaseEpoch = v19 起 QA 周期的起点（续期过的扇区大于 Activation）；
	// FullQaPower = lotus miner.SectorIsFullQaPower 的判定结果（带 FULL 标志，或 VDW 覆盖整个
	// exp-PowerBaseEpoch 周期）。
	// Flags 用 uint64 而不是 v19 的 miner.SectorOnChainInfoFlags：model 层不绑定 actor 版本，
	// JSON 里就是一个数字，filscan 侧也按 uint64 解码。
	Flags          uint64
	PowerBaseEpoch abi.ChainEpoch
	FullQaPower    bool
}
