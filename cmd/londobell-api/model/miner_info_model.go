package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type MinersInfoRes struct {
	ID                   string         `bson:"ID" json:"ID"`
	Epoch                abi.ChainEpoch `bson:"epoch" json:"epoch"`
	Miner                string         `bson:"miner" json:"miner"`
	Owner                string         `bson:"owner" json:"owner"`
	Worker               string         `bson:"worker" json:"worker"`
	Controllers          []string       `bson:"controllers" json:"controllers"`
	Power                string         `bson:"power" json:"power"`
	QualityPower         string         `bson:"quality_power" json:"quality_power"`
	Balance              string         `bson:"balance" json:"balance"`
	AvailableBalance     string         `bson:"available_balance" json:"available_balance"`
	VestingFunds         string         `bson:"vesting_funds" json:"vesting_funds"`
	FeeDebt              string         `bson:"fee_debt" json:"fee_debt"`
	SectorSize           abi.SectorSize `bson:"sector_size" json:"sector_size"`
	SectorCount          uint64         `bson:"sector_count" json:"sector_count"`
	FaultSectorCount     uint64         `bson:"fault_sector_count" json:"fault_sector_count"`
	ActiveSectorCount    uint64         `bson:"active_sector_count" json:"active_sector_count"`
	LiveSectorSector     uint64         `bson:"live_sector_count" json:"live_sector_count"`
	RecoverSectorCount   uint64         `bson:"recover_sector_count" json:"recover_sector_count"`
	TerminateSectorCount uint64         `bson:"terminate_sector_count" json:"terminate_sector_count"`
	PrecommitSectorCount uint64         `bson:"precommit_sector_count" json:"precommit_sector_count"`
	InitialPledge        string         `bson:"initial_pledge" json:"initial_pledge"`
	PreCommitDeposits    string         `bson:"pre_commit_deposits" json:"pre_commit_deposits"`
	States               interface{}    `bson:"states" json:"states"`
}
