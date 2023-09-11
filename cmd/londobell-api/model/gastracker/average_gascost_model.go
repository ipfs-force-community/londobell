package gastracker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type AverageGasCostRes struct {
	Epoch                     int64
	AverageGasLimit           int64
	AverageGasFeeCap          primitive.Decimal128
	AverageGasPremium         primitive.Decimal128
	AverageGasUsed            int64
	AverageBaseFeeBurn        primitive.Decimal128
	AverageOverEstimationBurn primitive.Decimal128
	AverageMinerPenalty       primitive.Decimal128
	AverageMinerTip           primitive.Decimal128
}

type AverageGasCostListRes struct {
	AverageGasCosts []AverageGasCostRes
}

func NewMockAverageGasCostListRes() AverageGasCostListRes {
	return AverageGasCostListRes{
		[]AverageGasCostRes{
			{
				Epoch:                     123,
				AverageGasLimit:           123,
				AverageGasFeeCap:          model.MockDecimal128,
				AverageGasPremium:         model.MockDecimal128,
				AverageGasUsed:            123,
				AverageBaseFeeBurn:        model.MockDecimal128,
				AverageOverEstimationBurn: model.MockDecimal128,
				AverageMinerPenalty:       model.MockDecimal128,
				AverageMinerTip:           model.MockDecimal128,
			},
			{
				Epoch:                     234,
				AverageGasLimit:           234,
				AverageGasFeeCap:          model.MockDecimal128,
				AverageGasPremium:         model.MockDecimal128,
				AverageGasUsed:            234,
				AverageBaseFeeBurn:        model.MockDecimal128,
				AverageOverEstimationBurn: model.MockDecimal128,
				AverageMinerPenalty:       model.MockDecimal128,
				AverageMinerTip:           model.MockDecimal128,
			},
		},
	}
}

func (m AverageGasCostListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
