package lotusCmdModel

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
)

type StateMarketBalanceReq struct {
	Addr  string `json:"addr"`
	Epoch int64  `json:"epoch"`
}

type StateMarketBalanceRes struct {
	Addr          address.Address   `json:"addr"`
	Epoch         abi.ChainEpoch    `json:"epoch"`
	MarketBalance api.MarketBalance `json:"market_balance"`
}
