package model

import "github.com/filecoin-project/go-state-types/abi"

type TransferMessage struct {
	Cid     string
	RootCid string
	Epoch   abi.ChainEpoch
	From    string
	To      string
	Value   string
	Method  string
	Depth   int
	IsBlock bool
}

type TransferMessageIMToken struct {
	Cid     string
	RootCid string
	Epoch   abi.ChainEpoch
	From    string
	To      string
	Value   string
	Method  string
}

type TransferMessagesRes struct {
	TotalCount       int64             `json:"totalCount"`
	TransferMessages []TransferMessage `json:"transferMessages"`
}

type TransferMessagesIMTokenRes struct {
	TotalCount       int64                    `json:"totalCount"`
	TransferMessages []TransferMessageIMToken `json:"transferMessages"`
}
