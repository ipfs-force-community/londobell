package model

import "github.com/filecoin-project/go-state-types/abi"

type BlockHeader struct {
	BlockCid      string `bson:"_id" json:"_id"`
	Miner         string
	Epoch         abi.ChainEpoch
	ElectionProof interface{}
	Ticket        interface{}
	MessageCount  int
	Timestamp     int64
	Parents       []string
	FirstSeen     int64
	ParentWeight  string
}

type BlockHeaderRes struct {
	TotalCount   int64         `json:"totalCount"`
	BlockHeaders []BlockHeader `json:"blockHeaders"`
}
