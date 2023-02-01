package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AggProNetFeeRes struct {
	Cid         string
	Epoch       abi.ChainEpoch
	AggFee      primitive.Decimal128
	MethodName  string
	Miner       string
	SectorCount int
	BaseFee     primitive.Decimal128
	BlockTime   int64
}
