package model

import (
	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/go-state-types/abi"
	"go.uber.org/zap"
)

var (
	_ common.Document      = (*MinerSectorSummary)(nil)
	_ common.DetailPrinter = (*MinerSectorSummary)(nil)

	minerSectorSummaryColName    = getColName(MinerSectorSummary{})
	minerSectorSummaryEpochField = extractEpochFieldName(MinerSectorSummary{})
)

// MinerSectorSummaryDetail contains the summaries in days of miner sectors
type MinerSectorSummaryDetail struct {
	Summaies []*MinerSectorSummaryRange
}

// MinerSectorSummaryRange is the summary of sectors with remain duration within the range [Lower, Upper)
type MinerSectorSummaryRange struct {
	LowerBound              abi.ChainEpoch
	UpperBound              abi.ChainEpoch
	SectorCount             uint64
	DealCount               uint64
	TotalDealWeight         abi.DealWeight
	TotalVerifiedDealWeight abi.DealWeight
	TotalInitialPledge      abi.TokenAmount
}

// MinerSectorSummary shows the distribution of sector lifetimes of a miner
type MinerSectorSummary struct {
	ActorStateExBasic `bson:",inline"`
	Detail            MinerSectorSummaryDetail
}

// CollectionName impl common.Document
func (m *MinerSectorSummary) CollectionName() string {
	return minerSectorSummaryColName
}

// EpochField impl common.Document
func (m *MinerSectorSummary) EpochField() *string {
	return &minerSectorSummaryEpochField
}

// ResetPolicy impl common.Document
func (m *MinerSectorSummary) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(minerSectorSummaryEpochField, lower, upper), true
}

// PrintDetail impl common.DetailPrinter
func (m *MinerSectorSummary) PrintDetail(l *zap.SugaredLogger) {
	l.Infof("Basic: %#v", m.ActorStateExBasic)
	for si := range m.Detail.Summaies {
		l.Infof("\tSummary #%d: %#v", si, m.Detail.Summaies[si])
	}
}
