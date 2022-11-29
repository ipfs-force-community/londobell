package lotusCmdModel

import "github.com/ipfs/go-cid"

type ChainGetMessageReq struct {
	MessageCid string `json:"message_cid"`
}

type ChainGetMessageRes struct {
	MessageCid cid.Cid     `json:"message_cid"`
	Message    interface{} `json:"message"`
}
