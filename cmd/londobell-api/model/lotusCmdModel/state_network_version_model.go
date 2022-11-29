package lotusCmdModel

import (
	"github.com/filecoin-project/go-state-types/abi"
	apitypes "github.com/filecoin-project/lotus/api/types"
)

type StateNetworkVersionReq struct {
	Epoch int64 `json:"epoch"`
}

type StateNetworkVersionRes struct {
	Epoch          abi.ChainEpoch          `json:"epoch"`
	NetworkVersion apitypes.NetworkVersion `json:"network_version"`
}
