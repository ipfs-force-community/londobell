package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BlockMessage struct {
	From       string             `bson:"from" json:"From"`
	To         string             `bson:"to" json:"To"`
	Method     string             `bson:"method" json:"Method"`
	Value      string             `bson:"value" json:"Value"`
	Params     interface{}        `bson:"params" json:"Params"`
	SignedCid  string             `bson:"signed_cid" json:"SignedCid"`
	GasUsed    string             `bson:"gas_used" json:"GasUsed"`
	BlockTime  primitive.DateTime `bson:"block_time" json:"BlockTime"`
	Epoch      abi.ChainEpoch     `bson:"epoch" json:"Epoch"`
	ExitCode   exitcode.ExitCode  `bson:"exit_code" json:"ExitCode"`
	Nonce      uint64             `bson:"nonce" json:"Nonce"`
	Return     interface{}        `bson:"return" json:"Return"`
	GasLimit   int64              `bson:"gas_limit" json:"GasLimit"`
	GasPremium string             `bson:"gas_premium" json:"GasPremium"`
	GasFeeCap  string             `bson:"gas_fee_cap" json:"GasFeeCap"`
	Version    uint64             `bson:"version" json:"Version"`
	GasCost    interface{}        `bson:"gascost" json:"GasCost"`
}

type BlockMessagesRes struct {
	TotalCount    int64          `json:"totalCount"`
	BlockMessages []BlockMessage `json:"blockMessages"`
}
