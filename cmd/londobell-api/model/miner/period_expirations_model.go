package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type PeriodExpirationsRes struct {
	Expirations []Sector
}

//func NewMockPeriodExpirationsRes() PeriodExpirationsRes {
//	return PeriodExpirationsRes{
//		Expirations: []Sector{
//			{
//				Miner:              "1111",
//				SectorNumber:       10,
//				Activation:         10,
//				Expiration:         10,
//				InitialPledge:      model.MockDecimal128,
//				DealWeight:         model.MockDecimal128,
//				VerifiedDealWeight: model.MockDecimal128,
//				DealIDs:            []int64{1, 2, 3},
//				SimpleQaPower:      false,
//			},
//			{
//				Miner:              "1111",
//				SectorNumber:       11,
//				Activation:         10,
//				Expiration:         10,
//				InitialPledge:      model.MockDecimal128,
//				DealWeight:         model.MockDecimal128,
//				VerifiedDealWeight: model.MockDecimal128,
//				DealIDs:            []int64{4, 5, 6},
//				SimpleQaPower:      true,
//			},
//		},
//	}
//}

func (m PeriodExpirationsRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
