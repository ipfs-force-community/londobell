package miner

import (
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type PeriodWinCountsRes struct {
	WinCounts  int64
	GasRewards primitive.Decimal128
}

func NewEmptyPeriodWinCountsRes() PeriodWinCountsRes {
	return PeriodWinCountsRes{
		WinCounts:  0,
		GasRewards: common.EmptyDecimal,
	}
}

func NewMockPeriodWinCountsRes() PeriodWinCountsRes {
	return PeriodWinCountsRes{
		WinCounts:  10,
		GasRewards: model.MockDecimal128}
}

func (m PeriodWinCountsRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
