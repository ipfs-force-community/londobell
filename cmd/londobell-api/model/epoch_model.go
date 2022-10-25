package model

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type EpochReq struct {
	Epoch int64 `json:"epoch"`
}

type EpochRes struct {
	Cids            []cid.Cid        `json:"cids"`
	Parents         types.TipSetKey  `json:"parents"`
	Epoch           abi.ChainEpoch   `json:"epoch"`
	BlockTime       time.Time        `json:"block_time"`
	BlockCount      int              `json:"block_count"`
	WinCount        int64            `json:"win_count"`
	NetPower        abi.StoragePower `json:"net_power"`
	NetQualityPower abi.StoragePower `json:"net_quality_power"`
	NetRewards      big.Int          `json:"net_rewards"`
	BaseFee         abi.TokenAmount  `json:"base_fee"`
	Source          string           `json:"source"`
}
