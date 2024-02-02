package aggregators

import (
	"encoding/json"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

var EpochByMessageCidAggregator = `
[
    {
        $match: {
            Cid: {$eq: ctx.Cid}
        }
    }
]
`

func GetBlocksForMessage(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var epoch int64
	alog := log.With("method", "GetBlocksForMessage")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	{
		epochPipe, err := util.Parse(model.Ctx{Cid: req.Cid}, EpochByMessageCidAggregator)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
		var msgEpoch model.EpochReq
		multiResult, err := multiquery.MultiTraversalQuery(ctx, epochPipe, countUtils, "ExecTrace")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}
		rawByte, err := json.Marshal(multiResult[0])
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &msgEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
		epoch = msgEpoch.Epoch

	}

	pipe, err := util.Parse(model.Ctx{Cid: req.Cid, StartEpoch: epoch}, string(common.BlocksForMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var blocksForMessageRes []model.BlockHeader

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "BlockMessage")
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

		err = json.Unmarshal(rawByte, &blocksForMessageRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = blocksForMessageRes
	c.JSON(http.StatusOK, res)
}
