package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type MinersInfoRes struct {
	ID                   string         `bson:"ID" json:"ID"`
	Epoch                abi.ChainEpoch `bson:"epoch" json:"Epoch"`
	Miner                string         `bson:"miner" json:"Miner"`
	Owner                string         `bson:"owner" json:"Owner"`
	Worker               string         `bson:"worker" json:"Worker"`
	Controllers          []string       `bson:"controllers" json:"ControlAddresses"`
	Power                string         `bson:"power" json:"RawBytePower"`
	QualityPower         string         `bson:"quality_power" json:"QualityAdjPower"`
	Balance              string         `bson:"balance" json:"Balance"`
	AvailableBalance     string         `bson:"available_balance" json:"AvailableBalance"`
	VestingFunds         string         `bson:"vesting_funds" json:"VestingFunds"`
	FeeDebt              string         `bson:"fee_debt" json:"FeeDebt"`
	SectorSize           abi.SectorSize `bson:"sector_size" json:"SectorSize"`
	SectorCount          uint64         `bson:"sector_count" json:"SectorCount"`
	FaultSectorCount     uint64         `bson:"fault_sector_count" json:"FaultSectorCount"`
	ActiveSectorCount    uint64         `bson:"active_sector_count" json:"ActiveSectorCount"`
	LiveSectorSector     uint64         `bson:"live_sector_count" json:"LiveSectorCount"`
	RecoverSectorCount   uint64         `bson:"recover_sector_count" json:"RecoverSectorCount"`
	TerminateSectorCount uint64         `bson:"terminate_sector_count" json:"TerminateSectorCount"`
	PrecommitSectorCount uint64         `bson:"precommit_sector_count" json:"PreCommitSectorCount"`
	InitialPledge        string         `bson:"initial_pledge" json:"InitialPledge"`
	PreCommitDeposits    string         `bson:"pre_commit_deposits" json:"PreCommitDeposits"`
	States               interface{}    `bson:"states" json:"State"`
}
