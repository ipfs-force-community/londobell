package model

import "github.com/filecoin-project/go-state-types/abi"

type MessageForCreate struct {
	ID      string `bson:"_id" json:"_id"`
	Cid     string
	Epoch   abi.ChainEpoch
	From    string
	To      string
	Value   string
	Method  string
	Caller  string // construtor caller
	ActorID string //CreateExternal Created
}

type MessagesForCreateRes struct {
	TotalCount        int64              `json:"totalCount"`
	MessagesForCreate []MessageForCreate `json:"messagesForCreate"`
}
