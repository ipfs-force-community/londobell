package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BlockMessage struct {
	From       string
	To         string
	Method     string
	Value      string
	Params     interface{}
	SignedCid  string
	GasUsed    string
	BlockTime  primitive.DateTime
	Epoch      abi.ChainEpoch
	ExitCode   exitcode.ExitCode
	Nonce      uint64
	Return     interface{}
	GasLimit   int64
	GasPremium string
	GasFeeCap  string
	Version    uint64
	GasCost    interface{}
}

type BlockMessagesRes struct {
	TotalCount    int64          `json:"totalCount"`
	BlockMessages []BlockMessage `json:"blockMessages"`
}
