package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type SectorHealthStats struct {
	AllSectors    int64
	LiveSectors   int64
	LiveQAPower   primitive.Decimal128
	LiveRawPower  primitive.Decimal128
	FaultSectors  int64
	FaultQAPower  primitive.Decimal128
	FaultRawPower primitive.Decimal128
}

func NewMockSectorHealthStats() SectorHealthStats {
	return SectorHealthStats{
		AllSectors:    123,
		LiveSectors:   123,
		LiveQAPower:   model.MockDecimal128,
		LiveRawPower:  model.MockDecimal128,
		FaultSectors:  123,
		FaultQAPower:  model.MockDecimal128,
		FaultRawPower: model.MockDecimal128,
	}
}

func (m SectorHealthStats) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
