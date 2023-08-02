package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type SectorRes struct {
	Sectors []Sector
}
type Sector struct {
	Miner                 string
	Epoch                 int64
	SectorNumber          int64
	SealProof             int64
	SealedCID             string
	DealIDs               []int64
	Activation            int64
	Expiration            int64
	DealWeight            string
	VerifiedDealWeight    string
	InitialPledge         string
	ExpectedDayReward     string
	ExpectedStoragePledge string
	ReplacedSectorAge     int64
	ReplacedDayReward     string // todo: 可能为null
	SectorKeyCID          string // todo: 可能为null
	SimpleQAPower         bool
	Added                 bool
	Removed               bool
}

//func NewMockSectorRes() SectorRes {
//	return SectorRes{
//		[]Sector{
//			{
//				Epoch:              123,
//				Miner:              "123",
//				SectorNumber:       123,
//				Activation:         123,
//				Expiration:         123,
//				InitialPledge:      model.MockDecimal128,
//				DealWeight:         model.MockDecimal128,
//				VerifiedDealWeight: model.MockDecimal128,
//				DealIDs:            []int64{1, 2, 3},
//				SimpleQaPower:      false,
//			},
//			{
//				Epoch:              123,
//				Miner:              "234",
//				SectorNumber:       234,
//				Activation:         234,
//				Expiration:         234,
//				InitialPledge:      model.MockDecimal128,
//				DealWeight:         model.MockDecimal128,
//				VerifiedDealWeight: model.MockDecimal128,
//				DealIDs:            []int64{4, 5, 6},
//				SimpleQaPower:      true,
//			},
//		},
//	}
//}

func (m SectorRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
