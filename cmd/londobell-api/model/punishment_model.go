package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PunishmentRes struct {
	Miner       string               `bson:"miner" json:"Miner"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"Epoch"`
	BlockTime   primitive.DateTime   `bson:"block_time" json:"BlockTime"`
	Value       primitive.Decimal128 `bson:"value" json:"Value"`
	PenaltyType string               `bson:"penalty_type" json:"PenaltyType"`
	Source      string               `bson:"source" json:"Source"`
}
