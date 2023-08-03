package aggregators

import (
	"encoding/json"
	"net/http"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetChildEpoch(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetChildEpoch")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var childEpochRes []model.ChildEpochRes
	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.StartEpoch+1, countUtils, childEpochAggregator, req, "Tipset")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &childEpochRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	if len(childEpochRes) == 0 || childEpochRes[0].ChildEpoch == 0 {
		// multi dbs query
		{
			multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.StartEpoch+1, countUtils, childEpoch2Aggregator, req, "Tipset")
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

			err = json.Unmarshal(rawByte, &childEpochRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}

		res.Data = childEpochRes
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = childEpochRes
	c.JSON(http.StatusOK, res)
}
