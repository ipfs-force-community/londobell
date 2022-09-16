package model

type WinCountRes struct {
	Miner         string `bson:"_id" json:"_id"`
	TotalWinCount int64  `bson:"totalWincount" json:"TotalWinCount"`
}
