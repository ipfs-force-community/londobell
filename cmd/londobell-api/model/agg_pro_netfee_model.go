package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AggProNetfeeRes struct {
	Cid         string               `bson:"cid" json:"cid"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"epoch"`
	AggFee      primitive.Decimal128 `bson:"aggFee" json:"aggFee"`
	MethodName  string               `bson:"methodName" json:"methodName"`
	Miner       string               `bson:"miner" json:"miner"`
	SectorCount int                  `bson:"sectorCount" json:"sectorCount"`
	BaseFee     primitive.Decimal128 `bson:"baseFee" json:"baseFee"`
	BlockTime   int64                `bson:"blockTime" json:"blockTime"`
}
