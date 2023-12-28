package aggregators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	rmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func GetFilSupply(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetFilSupply")
	req := model.FilSupplyReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)

	var filSupplyRes []rmodel.FilSupply

	pipeJS := `
	[
		{
			$match: {
				"_id": {$in: %s},
			}
		},
		{
			$sort: {
				"_id": -1
			}
		}
	]`

	epochsStr := fmt.Sprintf("%v", req.Epochs)
	epochsStr = strings.Trim(epochsStr, "[]")
	epochsStr = strings.ReplaceAll(epochsStr, " ", ",")
	epochsStr = fmt.Sprintf("[" + epochsStr + "]")

	pipe, err := util.Parse(model.Ctx{}, fmt.Sprintf(pipeJS, epochsStr))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	multiResult, err := multiquery.MultiUnionQuery(ctx, pipe, countUtils, "FilSupply")
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if len(multiResult) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	raw := multiResult
	rawByte, err := json.Marshal(raw)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	err = json.Unmarshal(rawByte, &filSupplyRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = filSupplyRes
	c.JSON(http.StatusOK, res)
}
