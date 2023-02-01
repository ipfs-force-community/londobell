package model

import "github.com/filecoin-project/go-state-types/abi"

type MessageForActor struct {
	Cid      string
	Epoch    abi.ChainEpoch
	From     string
	To       string
	Value    string
	ExitCode int64
	Method   string
}

type MessagesForActorRes struct {
	TotalCount       int64             `json:"totalCount"`
	MessagesForActor []MessageForActor `json:"messagesForActor"`
}
