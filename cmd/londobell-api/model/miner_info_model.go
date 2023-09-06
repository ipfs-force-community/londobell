package model

import (
	"github.com/filecoin-project/go-state-types/abi"
)

type MinersInfoRes struct {
	ID                     string
	Epoch                  abi.ChainEpoch
	Miner                  string
	Owner                  string
	Worker                 string
	ControlAddresses       []string
	RawBytePower           string
	QualityAdjPower        string
	Balance                string
	AvailableBalance       string
	VestingFunds           string
	FeeDebt                string
	SectorSize             abi.SectorSize
	SectorCount            uint64
	FaultSectorCount       uint64
	ActiveSectorCount      uint64
	LiveSectorSector       uint64
	RecoverSectorCount     uint64
	TerminateSectorCount   uint64
	PreCommitSectorCount   uint64
	InitialPledge          string
	PreCommitDeposits      string
	Beneficiary            string
	BeneficiaryTerm        interface{}
	PendingBeneficiaryTerm interface{}
	States                 interface{}
	Multiaddrs             interface{}
	PeerID                 interface{}
	UnprovenSectorCount    uint64
}
