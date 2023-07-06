package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TraceRes struct {
	ID           string //`bson:"_id" json:"_id"`
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
	ReturnBson   primitive.Binary
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
	ParamsBson   primitive.Binary
	Params       interface{}
	Detail       interface{}
	Actor        string
	IsBlock      bool
}
