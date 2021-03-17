package grafana

import "github.com/filecoin-project/go-state-types/abi"

type point struct {
	Epoch abi.ChainEpoch
	Value float64
}

type searchReq struct {
	Target string `json:"target"`
}

type searchResp []string
