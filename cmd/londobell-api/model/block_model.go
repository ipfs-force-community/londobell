package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BlockRes struct {
	From       string             `bson:"from" json:"from"`
	To         string             `bson:"to" json:"to"`
	Method     string             `bson:"method" json:"method"`
	Value      string             `bson:"value" json:"value"`
	Params     interface{}        `bson:"params" json:"params"`
	SignedCid  string             `bson:"signed_cid" json:"signed_cid"`
	GasUsed    string             `bson:"gas_used" json:"gas_used"`
	BlockTime  primitive.DateTime `bson:"block_time" json:"block_time"`
	Epoch      abi.ChainEpoch     `bson:"epoch" json:"epoch"`
	ExitCode   exitcode.ExitCode  `bson:"exit_code" json:"exit_code"`
	Nonce      uint64             `bson:"nonce" json:"nonce"`
	Return     interface{}        `bson:"return" json:"return"`
	GasLimit   int64              `bson:"gas_limit" json:"gas_limit"`
	GasPremium string             `bson:"gas_premium" json:"gas_premium"`
	GasFeeCap  string             `bson:"gas_fee_cap" json:"gas_fee_cap"`
	Version    uint64             `bson:"version" json:"version"`
	GasCost    interface{}        `bson:"gascost" json:"gascost"`
}
