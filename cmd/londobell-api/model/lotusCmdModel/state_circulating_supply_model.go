package lotusCmdModel

import "github.com/filecoin-project/go-state-types/abi"

type StateCirculatingSupplyReq struct {
	Epoch int64 `json:"epoch"`
}

type StateCirculatingSupplyRes struct {
	Epoch             abi.ChainEpoch  `json:"epoch"`
	CirculatingSupply abi.TokenAmount `json:"circulating_supply"`
}
