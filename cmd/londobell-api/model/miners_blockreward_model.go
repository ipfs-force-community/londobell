package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MinersBlockRewardRes struct {
	ID               interface{}          `bson:"_id" json:"_id"`
	TotalBlockReward primitive.Decimal128 `bson:"totalBlockReward" json:"TotalBlockReward"`
	BlockCount       int                  `bson:"blockcount" json:"BlockCount"`
}
