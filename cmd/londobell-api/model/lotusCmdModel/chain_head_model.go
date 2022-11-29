package lotusCmdModel

import "github.com/filecoin-project/lotus/chain/types"

type ChainHeadReq struct {
}

type ChainHeadRes struct {
	Head *types.TipSet `json:"head"`
}
