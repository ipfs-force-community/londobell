package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetTransferMessages(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransferMessages")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	req.Addr, err = GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetTotalCountForActorTransferMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.Count
	}

	var transferMessages []model.TransferMessage
	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, countUtils, transferMsgsForActorAggregator, req, "ActorMessage")
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

		err = json.Unmarshal(rawByte, &transferMessages)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.TransferMessagesRes{TotalCount: totalCount, TransferMessages: transferMessages}
	c.JSON(http.StatusOK, res)
}
