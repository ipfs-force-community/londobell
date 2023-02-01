package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PunishmentRes struct {
	Miner       string
	Epoch       abi.ChainEpoch
	BlockTime   primitive.DateTime
	Value       string
	PenaltyType string
	Source      string
}
