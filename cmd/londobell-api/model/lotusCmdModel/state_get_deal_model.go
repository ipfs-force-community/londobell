package lotusCmdModel

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
)

type StateGetDealReq struct {
	Epoch  int64  `json:"epoch"`
	DealID string `json:"dealId"`
}

type StateGetDealRes struct {
	Epoch  abi.ChainEpoch  `json:"epoch"`
	DealID abi.DealID      `json:"dealId"`
	Deal   *api.MarketDeal `json:"deal"`
}
