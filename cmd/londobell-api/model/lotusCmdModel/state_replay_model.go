package lotusCmdModel

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/ipfs/go-cid"
)

type StateReplayReq struct {
	MessageCid string `json:"message_cid"`
}

type StateReplayRes struct {
	MessageCid cid.Cid          `json:"message_cid"`
	Result     *api.InvocResult `json:"result"`
}
