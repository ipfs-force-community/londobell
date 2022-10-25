package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//todo:类型匹配
type AggPreNetfeeRes struct {
	Miner       string               `bson:"miner" json:"Miner"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"Epoch"`
	SectorCount int                  `bson:"sectorCount" json:"SectorCount"`
	SignedCid   string               `bson:"signedCid" json:"SignedCid"`
	MethodName  string               `bson:"methodName" json:"MethodName"`
	BaseFee     primitive.Decimal128 `bson:"baseFee" json:"BaseFee"`
	AggFee      primitive.Decimal128 `bson:"aggFee" json:"AggFee"`
	BlockTime   int64                `bson:"blockTime" json:"BlockTime"`
}
