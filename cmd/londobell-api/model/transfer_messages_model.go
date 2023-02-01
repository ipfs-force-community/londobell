package model

import "github.com/filecoin-project/go-state-types/abi"

type TransferMessage struct {
	Cid    string         `bson:"signed_cid" json:"Cid"`
	Epoch  abi.ChainEpoch `bson:"epoch" json:"Epoch"`
	From   string         `bson:"from" json:"From"`
	To     string         `bson:"to" json:"To"`
	Value  string         `bson:"value" json:"Value"`
	Method int            `bson:"method" json:"Method"`
}

type TransferMessagesRes struct {
	TotalCount       int64             `json:"totalCount"`
	TransferMessages []TransferMessage `json:"transferMessages"`
}
