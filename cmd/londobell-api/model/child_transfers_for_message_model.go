package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChildTransfersForMessageRes struct {
	Epoch        abi.ChainEpoch `json:"_id"`
	TransferList []Message
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
	ParamsBson   primitive.Binary
	ReturnsBson  primitive.Binary
	MethodNum    uint64
	Actor        string
}

type Message struct {
	Cid        string `bson:"_id" json:"_id"`
	Version    uint64
	To         string
	From       string
	Nonce      uint64
	Value      string
	GasLimit   int64
	GasFeeCap  string
	GasPremium string
	Method     uint64
	Params     interface{}
	Detail     interface{}
	SignedCid  string
}

type GasCost struct {
	Message            string
	GasUsed            string
	BaseFeeBurn        string
	OverEstimationBurn string
	MinerPenalty       string
	MinerTip           string
	Refund             string
	TotalCost          string
}
