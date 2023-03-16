package model

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*MinerFunds)(nil)

	minerFundsColName    = getColName(MinerFunds{})
	minerFundsEpochField = extractEpochFieldName(MinerFunds{})
)

// MinerFundsDetail contains several token amounts
type MinerFundsDetail struct {
	PreCommitDeposits abi.TokenAmount

	LockedFunds   abi.TokenAmount
	FeeDebt       abi.TokenAmount
	InitialPledge abi.TokenAmount

	VestInFuture  []abi.TokenAmount `mir:"-"`
	PledgeRelease []abi.TokenAmount `mir:"-"`
}

type WorkerKeyChange struct {
	NewWorker   addr.Address
	EffectiveAt abi.ChainEpoch
}

type BeneficiaryTerm struct {
	Quota      abi.TokenAmount
	UsedQuota  abi.TokenAmount
	Expiration abi.ChainEpoch
}

type PendingBeneficiaryChange struct {
	NewBeneficiary        addr.Address
	NewQuota              abi.TokenAmount
	NewExpiration         abi.ChainEpoch
	ApprovedByBeneficiary bool
	ApprovedByNominee     bool
}

type MinerInfo struct {
	Owner                      addr.Address
	Worker                     addr.Address
	ControlAddresses           []addr.Address
	PendingWorkerKey           WorkerKeyChange
	PeerID                     abi.PeerID
	Multiaddrs                 []abi.Multiaddrs
	WindowPoStProofType        abi.RegisteredPoStProof
	SectorSize                 abi.SectorSize
	WindowPoStPartitionSectors uint64
	ConsensusFaultElapsed      abi.ChainEpoch // 低版本没有？
	PendingOwnerAddress        *addr.Address  //
	Balance                    abi.TokenAmount
	AvailableBalance           abi.TokenAmount
	FeeDebt                    abi.TokenAmount
	PrecommitSectorCount       uint64
	Beneficiary                addr.Address              `mir:"-"`
	BeneficiaryTerm            BeneficiaryTerm           `mir:"-"`
	PendingBeneficiaryTerm     *PendingBeneficiaryChange `mir:"-"`
	State                      interface{}
}

// MinerFunds shows funding details for miner
type MinerFunds struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MinerFundsDetail
	Info              MinerInfo
}

// CollectionName impl common.Document
func (m *MinerFunds) CollectionName() string {
	return minerFundsColName
}

// EpochField impl common.Document
func (m *MinerFunds) EpochField() *string {
	return &minerFundsEpochField
}

// ResetPolicy impl common.Document
func (m *MinerFunds) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerFundsEpochField, lower, upper), true
}
