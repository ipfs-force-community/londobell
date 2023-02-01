package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type WinCountRes struct {
	Miner          string `bson:"_id" json:"_id"`
	TotalWinCount  int64
	TotalGasReward primitive.Decimal128
}
