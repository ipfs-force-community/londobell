package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MinerBlockRewardRes struct {
	Epoch            abi.ChainEpoch `bson:"_id" json:"_id"`
	TotalBlockReward primitive.Decimal128
	BlockCount       int
}
