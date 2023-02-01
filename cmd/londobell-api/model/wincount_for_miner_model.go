package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type WinCountForMiner struct {
	Miner     string
	WinCount  int64
	GasReward primitive.Decimal128
}

type WinCountForMinerRes struct {
	TotalWinCount int64
	//TotalGasReward primitive.Decimal128
}
