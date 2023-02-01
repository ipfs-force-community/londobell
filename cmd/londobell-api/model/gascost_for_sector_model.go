package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type GasCostForSectorRes struct {
	Miner   string               `bson:"_id" json:"Miner"`
	GasCost primitive.Decimal128 `bson:"gasCost" json:"GasCost"`
}
