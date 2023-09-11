package miner

import (
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PeriodBlockRewardsRes struct {
	BlockRewards primitive.Decimal128
	BlockCounts  int64
}

func NewEmptyPeriodBlockRewardsRes() PeriodBlockRewardsRes {
	return PeriodBlockRewardsRes{
		BlockRewards: common.EmptyDecimal,
		BlockCounts:  0,
	}
}
func NewMockPeriodBlockRewardsRes() PeriodBlockRewardsRes {
	return PeriodBlockRewardsRes{
		BlockRewards: model.MockDecimal128,
		BlockCounts:  10}
}

func (m PeriodBlockRewardsRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
