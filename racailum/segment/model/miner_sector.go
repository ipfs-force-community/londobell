package model

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                     common.IndexedDocument = (*MinerSector)(nil)
	minerSectorEpochField                        = extractEpochFieldName(MinerSector{})
	minerSectorColName                           = getColName(MinerSector{})
)

// 由新高度新增变化记录 --> update原记录
type MinerSector struct {
	ID                 string `bson:"_id"`
	Miner              address.Address
	SectorNumber       abi.SectorNumber
	DealIDs            []abi.DealID
	Activation         abi.ChainEpoch // 即创建高度
	Expiration         abi.ChainEpoch
	DealWeight         abi.DealWeight
	VerifiedDealWeight abi.DealWeight
	SimpleQaPower      bool
	InitialPledge      abi.TokenAmount
	Terminated         bool           // 不会被其他记录覆盖
	Epoch              abi.ChainEpoch // 分隔库，以最新高度为准， newSector: Epoch == Activation

	//// todo：其他字段需要的时候再单独查
	//Deadline           uint64
	//Partition          uint64

	// 零点读取一次，报警一周内将要过期的扇区
	// 如果用户此时续期，报警可能要等一天才会取消报警

	//SealProof             abi.RegisteredSealProof // The seal proof type implies the PoSt proof/s
	//SealedCID             cid.Cid                 // CommR
	//ExpectedDayReward     abi.TokenAmount         // Expected one day projection of reward for sector computed at activation time
	//ExpectedStoragePledge abi.TokenAmount         // Expected twenty day projection of reward for sector computed at activation time
	//ReplacedSectorAge     abi.ChainEpoch          // Age of sector this sector replaced or zero
	//ReplacedDayReward     abi.TokenAmount         // Day reward of sector this sector replace or zero
	//SectorKeyCID          *cid.Cid
}

// sector 出错报警: misspost、磁盘故障

func NewMinerSector(miner address.Address, sectorNumber abi.SectorNumber, dealIDs []abi.DealID, activation abi.ChainEpoch, expiration abi.ChainEpoch, dealWeight abi.DealWeight, verifiedDealWeight abi.DealWeight, simpleQaPower bool, initialPledge abi.TokenAmount, terminated bool, epoch abi.ChainEpoch) *MinerSector {
	return &MinerSector{
		ID:                 fmt.Sprintf("%v-%v", miner, sectorNumber),
		Miner:              miner,
		SectorNumber:       sectorNumber,
		DealIDs:            dealIDs,
		Activation:         activation,
		Expiration:         expiration,
		DealWeight:         dealWeight,
		VerifiedDealWeight: verifiedDealWeight,
		SimpleQaPower:      simpleQaPower,
		InitialPledge:      initialPledge,
		Terminated:         terminated,
		Epoch:              epoch,
	}
}

// CollectionName impl CollectionName
func (m *MinerSector) CollectionName() string {
	return minerSectorColName
}

// EpochField impl common.Document
func (m *MinerSector) EpochField() *string {
	return &minerSectorEpochField
}

// ResetPolicy impl common.Document
func (m *MinerSector) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerSectorEpochField, lower, upper), true
}

func (m *MinerSector) Indexes() [][]string {
	return [][]string{
		[]string{"Miner", "SectorNumber"},
		[]string{"Miner", "SimpleQaPower"},
		[]string{"Miner", "DealIDs"},
		[]string{"Miner", "Expiration"},
		[]string{minerSectorEpochField, "Activation", "Replaced"},
		[]string{"Miner", "Terminated"},
	}
}

func (m *MinerSector) IsMutable() bool {
	return true
}
