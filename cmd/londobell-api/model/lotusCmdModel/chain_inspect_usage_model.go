package lotusCmdModel

import "github.com/filecoin-project/go-state-types/abi"

type ChainInspectUsageReq struct {
	Epoch      int64 `json:"epoch"`
	Length     int   `json:"length" default:"1"`
	NumResults int   `json:"num-results"`
}

type ChainInspectUsageRes struct {
	Epoch     abi.ChainEpoch `json:"epoch"`
	Senders   []InspectUsage `json:"senders"`
	Receivers []InspectUsage `json:"receivers"`
	Methods   []InspectUsage `json:"methods"`
}

type InspectUsage struct {
	Key           string
	GasLimitRatio float64
	Total         int64
	Count         int64
}
