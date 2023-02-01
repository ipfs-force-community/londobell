package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type GasCostForSectorRes struct {
	Miner   string `bson:"_id" json:"_id"`
	GasCost primitive.Decimal128
}
