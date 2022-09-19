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
	Detail       interface{}
	GasCost      interface{}
}
