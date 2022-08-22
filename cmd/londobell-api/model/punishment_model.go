package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PunishmentRes struct {
	Miner       string               `bson:"miner" json:"miner"`
	Epoch       abi.ChainEpoch       `bson:"epoch" json:"epoch"`
	BlockTime   primitive.DateTime   `bson:"block_time" json:"block_time"`
	Value       primitive.Decimal128 `bson:"value" json:"value"`
	PenaltyType string               `bson:"penalty_type" json:"penalty_type"`
	Source      string               `bson:"source" json:"source"`
}
