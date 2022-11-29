package lotusCmdModel

import "github.com/filecoin-project/lotus/chain/types"

type ChainGasPricesReq struct {
}

type ChainGasPricesRes struct {
	NBlocks            int          `json:"n_blocks"`
	EstimateGasPremium types.BigInt `json:"estimate_gas_premium"`
}
