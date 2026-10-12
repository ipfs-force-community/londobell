package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	rmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func GetRichList(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetRichList")
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

	var richList []model.Rich

	// RichList needs the whole ranked set at one epoch. ActorBalance no longer
	// has such an epoch (every snapshot epoch writes only the actors that
	// changed), so the read moved to the daily top-N collection, which is
	// written once per day at the zero hour. The response shape is unchanged:
	// the endpoint still returns the same top 1000.
	snapEpoch, ok := actorBalanceTopSnapshotEpoch(req.StartEpoch)
	if !ok {
		alog.Warnf("no top-N ActorBalance snapshot at or before %d, querying that epoch alone", req.StartEpoch)
	}

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, snapEpoch, snapEpoch+1, countUtils, common2.RichListTopPipeline(common2.RichListAggregator), req, rmodel.ActorBalanceTopCollection)
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

		err = json.Unmarshal(rawByte, &richList)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.RichListRes{TotalCount: 1000, RichList: richList}
	c.JSON(http.StatusOK, res)
}
