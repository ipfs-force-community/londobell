package model

type MinedCountForMinersRes struct {
	Miner      string `bson:"_id" json:"Miner"`
	MinedCount int64  `bson:"minedCount" json:"MinedCount"`
}
