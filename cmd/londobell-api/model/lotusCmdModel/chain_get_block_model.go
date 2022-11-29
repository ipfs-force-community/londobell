package lotusCmdModel

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type ChainGetBlockReq struct {
	BlockCid string `json:"block_cid"`
}

type ChainGetBlockRes struct {
	BlockCid       cid.Cid                 `json:"block_cid"`
	BlockHeader    types.BlockHeader       `json:"block_header"`
	BlsMessages    []*types.Message        `json:"bls_messages"`
	SecpkMessages  []*types.SignedMessage  `json:"secpk_messages"`
	ParentReceipts []*types.MessageReceipt `json:"parent_receipts"`
	ParentMessages []cid.Cid               `json:"parent_messages"`
}
