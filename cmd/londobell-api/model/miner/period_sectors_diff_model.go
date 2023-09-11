package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type PeriodSectorsDiffRes struct {
	AllSectorsDiff    int64
	LiveSectorsDiff   int64
	LiveQAPowerDiff   primitive.Decimal128
	LiveRawPowerDiff  primitive.Decimal128
	FaultSectorsDiff  int64
	FaultQAPowerDiff  primitive.Decimal128
	FaultRawPowerDiff primitive.Decimal128
}

func NewMockPeriodSectorsDiffRes() PeriodSectorsDiffRes {
	return PeriodSectorsDiffRes{
		AllSectorsDiff:    10,
		LiveSectorsDiff:   10,
		LiveQAPowerDiff:   model.MockDecimal128,
		LiveRawPowerDiff:  model.MockDecimal128,
		FaultSectorsDiff:  10,
		FaultQAPowerDiff:  model.MockDecimal128,
		FaultRawPowerDiff: model.MockDecimal128,
	}
}

func (m PeriodSectorsDiffRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
