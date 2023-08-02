package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

//type PeriodGasCostsRes struct {
//	GasCosts []GasCost
//}

type GasCost struct {
	MethodName string `bson:"_id" json:"_id"`
	GasCosts   primitive.Decimal128
	Values     primitive.Decimal128
}

func GasCostToMap(res []GasCost) (map[string]GasCost, error) {
	gasCostsMap := make(map[string]GasCost)
	for _, gascost := range res {
		if _, ok := gasCostsMap[gascost.MethodName]; !ok {
			gasCostsMap[gascost.MethodName] = gascost
		} else {
			preGascost := gasCostsMap[gascost.MethodName]
			gasCosts, err := common2.AddDecimal128(preGascost.GasCosts, gascost.GasCosts)
			if err != nil {
				return nil, err
			}

			values, err := common2.AddDecimal128(preGascost.Values, gascost.Values)
			if err != nil {
				return nil, err
			}

			gasCostsMap[gascost.MethodName] = GasCost{MethodName: gascost.MethodName, GasCosts: gasCosts, Values: values}
		}
	}

	return gasCostsMap, nil
}

func NewNullGasCost() GasCost {
	return GasCost{}
}

func NewMockGasCosts() []GasCost {
	return []GasCost{
		{
			MethodName: "PreCommitSector",
			GasCosts:   model.MockDecimal128,
			Values:     model.MockDecimal128,
		},
		{
			MethodName: "ProveCommitSector",
			GasCosts:   model.MockDecimal128,
			Values:     model.MockDecimal128,
		},
	}
}

func (m GasCost) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = NewMockGasCosts()

	c.JSON(http.StatusOK, res)
}
