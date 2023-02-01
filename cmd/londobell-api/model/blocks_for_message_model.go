package model

import "github.com/filecoin-project/go-state-types/abi"

type BlocksForMessage struct {
	Cid    string `bson:"_id" json:"Cid"`
	Epoch  abi.ChainEpoch
	Blocks []string
}
