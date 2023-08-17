package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type TipSetRes struct {
	Epoch        abi.ChainEpoch `bson:"_id" json:"_id"`
	Cids         []string
	MinTimestamp uint64
	ChildEpoch   abi.ChainEpoch
	State        string
	Receipts     string
	Weight       string
	BaseFee      string
}

type TipSetListRes struct {
	TotalCount int64       `json:"totalCount"`
	TipSets    []TipSetRes `json:"tipSets"`
}
