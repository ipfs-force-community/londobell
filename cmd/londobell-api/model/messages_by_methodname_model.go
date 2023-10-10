package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
)

type MessageByMethodName struct {
	SignedCid string
	RootCid   string
	Epoch     abi.ChainEpoch
	From      string
	To        string
	Value     string
	ExitCode  exitcode.ExitCode
	Method    string
}

type MessagesByMethodNameRes struct {
	TotalCount           int64                 `json:"totalCount"`
	MessagesByMethodName []MessageByMethodName `json:"messagesByMethodName"`
}
