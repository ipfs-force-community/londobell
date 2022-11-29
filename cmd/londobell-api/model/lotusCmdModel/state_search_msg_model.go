package lotusCmdModel

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type StateSearchMsgReq struct {
	MessageCid string `json:"message_cid"`
}

type StateSearchMsgRes struct {
	MessageCid cid.Cid        `json:"message_cid"`
	Message    *types.Message `json:"message"`
}
