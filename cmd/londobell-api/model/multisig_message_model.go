package model

import "github.com/filecoin-project/go-state-types/abi"

type MultisigMessageRes struct {
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

	//Trace        lmodel.ExecTrace `bson:",inline" json:",inline"` //
	//Message      lmodel.Message   `bson:"message" json:"message"`
	//ChildTrace   lmodel.ExecTrace `bson:"childTrace" json:"childTrace"`
	//ChildMessage lmodel.Message   `bson:"childMessage" json:"childMessage"`

	//Trace        interface{} `bson:",inline" json:",inline"` //
	Message      interface{} `bson:"message" json:"message"`
	ChildTrace   interface{} `bson:"childTrace" json:"childTrace"`
	ChildMessage interface{} `bson:"childMessage" json:"childMessage"`
}
