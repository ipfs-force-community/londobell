package model

import "github.com/filecoin-project/go-state-types/abi"

type TransferMessage struct {
	Cid    string
	Epoch  abi.ChainEpoch
	From   string
	To     string
	Value  string
	Method string
	Depth  int
}

type TransferMessagesRes struct {
	TotalCount       int64             `json:"totalCount"`
	TransferMessages []TransferMessage `json:"transferMessages"`
}
