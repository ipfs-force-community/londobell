package lotusCmdModel

import "github.com/filecoin-project/lotus/chain/types"

type ChainListReq struct {
	Epoch int64 `json:"epoch"`
	Count int   `json:"count" default:"30"`
}

type ChainListRes struct {
	Tipsets []*types.TipSet `json:"tipsets"`
}
