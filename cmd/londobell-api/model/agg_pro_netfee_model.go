package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AggProNetfeeRes struct {
	Cid         string               `bson:"cid" json:"Cid"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"Epoch"`
	AggFee      primitive.Decimal128 `bson:"aggFee" json:"AggFee"`
	MethodName  string               `bson:"methodName" json:"MethodName"`
	Miner       string               `bson:"miner" json:"Miner"`
	SectorCount int                  `bson:"sectorCount" json:"SectorCount"`
	BaseFee     primitive.Decimal128 `bson:"baseFee" json:"BaseFee"`
	BlockTime   int64                `bson:"blockTime" json:"BlockTime"`
}
