package model

import "github.com/filecoin-project/go-state-types/abi"

type TraceRes struct {
	ID           string `bson:"_id" json:"_id"`
	Cid          string
	SignedCid    string
	Epoch        abi.ChainEpoch
	Seq          []int
	Depth        int
	Ver          string
	Msg          interface{}
	MsgRct       interface{}
	Error        string
	SeqIndex     [][]int
	SubCallCount int
	GasCost      interface{}
	Return       interface{}
	Version      uint64
	To           string
	From         string
	Nonce        uint64
	Value        string
	GasLimit     int64
	GasFeeCap    string
	GasPremium   string
	Method       uint64
	Params       interface{}
	Detail       interface{}
	Actor        string `json:"-"`
}
