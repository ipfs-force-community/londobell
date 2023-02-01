package model

import (
	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AggPreNetFeeRes struct {
	Miner       string
	Epoch       abi.ChainEpoch
	SectorCount int
	SignedCid   string
	MethodName  string
	BaseFee     primitive.Decimal128
	AggFee      primitive.Decimal128
	BlockTime   int64
}
