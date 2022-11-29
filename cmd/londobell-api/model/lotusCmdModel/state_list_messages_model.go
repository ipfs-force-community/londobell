package lotusCmdModel

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type StateListMessagesReq struct {
	To       string `json:"to"`
	From     string `json:"from"`
	ToHeight uint64 `json:"to_height"`
	Epoch    int64  `json:"epoch"`
	Cids     bool   `json:"cids"`
}

type StateListMessagesRes struct {
	Messages    []*types.Message `json:"messages"`
	MessageCids []cid.Cid        `json:"message-cids"`
}
