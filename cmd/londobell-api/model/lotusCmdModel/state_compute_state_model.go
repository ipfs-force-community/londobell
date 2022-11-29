package lotusCmdModel

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
)

type StateComputeStateReq struct {
	Epoch              int64  `json:"epoch"`
	VMHeight           uint64 `json:"vm_height"`
	ApplyMpoolMessages bool   `json:"apply_mpool_messages"`
}

type StateComputeStateRes struct {
	Epoch       abi.ChainEpoch          `json:"epoch"`
	StateOutPut *api.ComputeStateOutput `json:"state_out_put"`
}
