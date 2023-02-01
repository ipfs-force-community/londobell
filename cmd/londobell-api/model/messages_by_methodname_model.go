package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
)

type MessageByMethodName struct {
	Cid      string            `bson:"cid" json:"Cid"`
	Epoch    abi.ChainEpoch    `bson:"epoch" json:"Epoch"`
	From     string            `bson:"from" json:"From"`
	To       string            `bson:"to" json:"To"`
	Value    string            `bson:"value" json:"Value"`
	ExitCode exitcode.ExitCode `bson:"exit_code" json:"ExitCode"`
	Method   string            `bson:"method" json:"Method"`
}

type MessagesByMethodNameRes struct {
	TotalCount           int64                 `json:"totalCount"`
	MessagesByMethodName []MessageByMethodName `json:"messagesByMethodName"`
}
