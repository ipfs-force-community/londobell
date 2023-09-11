package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type PeriodPledgeDiffRes struct {
	InitialPledgeDiff     primitive.Decimal128
	PreCommitDepositsDiff primitive.Decimal128
	LockedFundsDiff       primitive.Decimal128
}

func NewMockPeriodPledgeDiffRes() PeriodPledgeDiffRes {
	return PeriodPledgeDiffRes{
		InitialPledgeDiff:     model.MockDecimal128,
		PreCommitDepositsDiff: model.MockDecimal128,
		LockedFundsDiff:       model.MockDecimal128,
	}
}

func (m PeriodPledgeDiffRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
