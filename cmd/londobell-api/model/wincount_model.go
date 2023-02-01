package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type WinCountRes struct {
	Miner          string               `bson:"_id" json:"_id"`
	TotalWinCount  int64                `bson:"totalWincount" json:"TotalWinCount"`
	TotalGasReward primitive.Decimal128 `bson:"totalGasReward" json:"TotalGasReward"`
}
