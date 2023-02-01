package model

import "github.com/filecoin-project/go-state-types/abi"

type TraceForMessageRes struct {
	Cid          string
	Epoch        abi.ChainEpoch
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
	GasCost      GasCost
}
