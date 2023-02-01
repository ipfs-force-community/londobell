package model

import "github.com/filecoin-project/go-state-types/abi"

type MessageForActor struct {
	Cid      string         `bson:"signed_cid" json:"Cid"`
	Epoch    abi.ChainEpoch `bson:"epoch" json:"Epoch"`
	From     string         `bson:"from" json:"From"`
	To       string         `bson:"to" json:"To"`
	Value    string         `bson:"value" json:"Value"`
	ExitCode int64          `bson:"exit_code" json:"ExitCode"`
	Method   string         `bson:"method" json:"Method"`
}

type MessagesForActorRes struct {
	TotalCount       int64             `json:"totalCount"`
	MessagesForActor []MessageForActor `json:"messagesForActor"`
}
