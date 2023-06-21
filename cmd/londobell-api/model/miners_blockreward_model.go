package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MinersBlockRewardRes struct {
	ID               interface{} `bson:"_id" json:"_id"`
	TotalBlockReward primitive.Decimal128
	BlockCount       int
}
