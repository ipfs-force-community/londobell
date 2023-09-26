package model

import "github.com/filecoin-project/go-state-types/abi"

type ChildCallsForMessageRes struct {
	Epoch        abi.ChainEpoch `json:"_id"`
	InnerCalls   []InnerCall
	GasCost      GasCost
	Cid          string
	Value        string
	From         string
	To           string
	ExitCode     int64
	Method       string
	Params       interface{}
	Return       interface{}
	ParamsDetail interface{}
	ReturnDetail interface{}
	Version      uint64
	Nonce        uint64
	GasLimit     int64
	GasFeeCap    string
	GasPremium   string
	Error        string
}

type InnerCall struct {
	To         string
	From       string
	Value      string
	MethodName string
}
