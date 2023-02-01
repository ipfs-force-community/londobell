package model

import "github.com/filecoin-project/go-state-types/abi"

type TransferMessageForLargeAmount struct {
	Cid    string         `bson:"signed_cid" json:"Cid"`
	Epoch  abi.ChainEpoch `bson:"epoch" json:"Epoch"`
	From   string         `bson:"from" json:"From"`
	To     string         `bson:"to" json:"To"`
	Value  string         `bson:"value" json:"Value"`
	Method string         `bson:"method" json:"Method"`
}

type TransferMessagesForLargeAmountRes struct {
	TotalCount                     int64                           `json:"totalCount"`
	TransferMessagesForLargeAmount []TransferMessageForLargeAmount `json:"transferMessagesForLargeAmount"`
}
