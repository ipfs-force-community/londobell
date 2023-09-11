package miner

import (
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type PeriodPunishmentsRes struct {
	Punishments primitive.Decimal128
}

func NewEmptyPeriodPunishmentsRes() PeriodPunishmentsRes {
	return PeriodPunishmentsRes{Punishments: common2.EmptyDecimal}
}

func NewMockPeriodPunishmentsRes() PeriodPunishmentsRes {
	return PeriodPunishmentsRes{
		Punishments: model.MockDecimal128,
	}
}

func (m PeriodPunishmentsRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
