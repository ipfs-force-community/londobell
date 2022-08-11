package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//todo:类型匹配
type AggPreNetfeeRes struct {
	Miner       string               `bson:"miner" json:"miner"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"epoch"`
	SectorCount int                  `bson:"sectorCount" json:"sectorCount"`
	SignedCid   string               `bson:"signedCid" json:"signedCid"`
	MethodName  string               `bson:"methodName" json:"methodName"`
	BaseFee     primitive.Decimal128 `bson:"baseFee" json:"baseFee"`
	AggFee      primitive.Decimal128 `bson:"aggFee" json:"aggFee"`
	BlockTime   int64                `bson:"blockTime" json:"blockTime"`
}
