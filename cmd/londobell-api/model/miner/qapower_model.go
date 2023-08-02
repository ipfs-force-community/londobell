package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type QAPowerRes struct {
	VDCPower primitive.Decimal128
	DCPower  primitive.Decimal128
	CCPower  primitive.Decimal128
}

func NewMockQAPowerRes() QAPowerRes {
	return QAPowerRes{
		VDCPower: model.MockDecimal128,
		DCPower:  model.MockDecimal128,
		CCPower:  model.MockDecimal128,
	}
}

func (m QAPowerRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
