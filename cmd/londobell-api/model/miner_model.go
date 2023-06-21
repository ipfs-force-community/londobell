package model

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

type MinerReq struct {
	Miner string `json:"miner"`
	Epoch int64  `json:"epoch"`
}

type MinerRes struct {
	Epoch                    abi.ChainEpoch    `json:"epoch"`
	Miner                    address.Address   `json:"miner"`
	Owner                    address.Address   `json:"owner"`
	Worker                   address.Address   `json:"worker"`
	Controllers              []address.Address `json:"controllers"`
	SectorSize               abi.SectorSize    `json:"sector_size"`
	Power                    abi.StoragePower  `json:"power"`
	QualityPower             abi.StoragePower  `json:"quality_power"`
	Balance                  types.BigInt      `json:"balance"`
	AvailableBalance         types.BigInt      `json:"available_balance"`
	VestingFunds             abi.TokenAmount   `json:"vesting_funds"`
	LockedFunds              abi.TokenAmount   `json:"locked_funds"`
	InitialPledgeRequirement abi.TokenAmount   `json:"initial_pledge_requirement"`
	State                    interface{}       `json:"state"`
	SectorCount              uint64            `json:"sector_count"`
	FaultSectorCount         uint64            `json:"fault_sector_count"`
	ActiveSectorCount        uint64            `json:"active_sector_count"`
	LiveSectorCount          uint64            `json:"live_sector_count"`
	RecoverSectorCount       uint64            `json:"recover_sector_count"`
	TerminateSectorCount     uint64            `json:"terminate_sector_count"`
	PrecommitSectorCount     uint64            `json:"precommit_sector_count"`
	Beneficiary              address.Address
	BeneficiaryTerm          *miner.BeneficiaryTerm
	PendingBeneficiaryTerm   *miner.PendingBeneficiaryChange
	PeerID                   *peer.ID
	Multiaddrs               []abi.Multiaddrs
}
