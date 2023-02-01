package model

import "github.com/filecoin-project/go-state-types/abi"

type TransferMessageForLargeAmount struct {
	Cid    string
	Epoch  abi.ChainEpoch
	From   string
	To     string
	Value  string
	Method string
}

type TransferMessagesForLargeAmountRes struct {
	TotalCount                     int64                           `json:"totalCount"`
	TransferMessagesForLargeAmount []TransferMessageForLargeAmount `json:"transferMessagesForLargeAmount"`
}
