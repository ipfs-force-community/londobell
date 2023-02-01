package model

import "github.com/filecoin-project/go-state-types/abi"

type ClaimedPowerRes struct {
	Miner  string `bson:"Addr" json:"Miner"`
	Epoch  abi.ChainEpoch
	Detail PowerDetail
}

type PowerDetail struct {
	RawBytePower    string
	QualityAdjPower string
}
