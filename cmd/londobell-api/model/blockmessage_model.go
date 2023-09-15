package model

import "github.com/filecoin-project/go-state-types/abi"

type BlockMessages struct {
	BlockCid string
	Epoch    abi.ChainEpoch
	Messages []string
}
