package miner

import (
	"context"
	"encoding/json"
	"net/http"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/miner"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetPeriodSectorExpirations(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("miner", "GetPeriodSectorExpirations")

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

	// only search from formal
	startEpoch, endEpoch := int64(0), req.EndEpoch
	for _, countUtil := range countUtils {
		if countUtil.DType == smodel.Formal {
			startEpoch = countUtil.Start
			if endEpoch > countUtil.End {
				endEpoch = countUtil.End
			}
			break
		}
	}

	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var periodSectorExpirationsRes []miner.Sector
	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, startEpoch, endEpoch, countUtils, common2.MinerPeriodSectorExpirationAggregator, req, "ChangedSector")
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

		err = json.Unmarshal(rawByte, &periodSectorExpirationsRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = periodSectorExpirationsRes
	c.JSON(http.StatusOK, res)
}
