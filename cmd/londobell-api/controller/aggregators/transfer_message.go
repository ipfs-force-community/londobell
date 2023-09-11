package aggregators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
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

	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var (
		countUtils []multiquery.CountUtil
		pipe       []byte
		countAgg   []byte
	)

	switch req.TransferType {
	case "":
		countUtils, err = multiquery.GetTotalCountForActorTransferMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferMsgsFroActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransfersForActor2Aggregator()
	case "blockreward":
		req.TransferType = "Blockreward"
		countUtils, err = multiquery.GetTotalCountForActorTransferBlockRewardMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferTypeForActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransferBlockRewardForActorAggregator()
	case "burn":
		req.TransferType = "Burn"
		countUtils, err = multiquery.GetTotalCountForActorTransferBurnMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferTypeForActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransferBurnForActorAggregator()
	case "transfer":
		countUtils, err = multiquery.GetTotalCountForActorTransferSendAndReceiveMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferSendAndReceiveForActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransferSendAndReceiveForActorAggregator()
	case "send":
		req.TransferType = "Send"
		countUtils, err = multiquery.GetTotalCountForActorTransferSendMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferTypeForActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransferSendForActorAggregator()
	case "receive":
		req.TransferType = "Receive"
		countUtils, err = multiquery.GetTotalCountForActorTransferReceiveMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe = common2.TransferTypeForActorNoSkipAggregator
		countAgg = monitor.GetCountOfTransferReceiveForActorAggregator()
	default:
		err = fmt.Errorf("invalid transfer type: %v", req.TransferType)
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.ActorTransferStates
	}

	var transferMessages []model.TransferMessage
	// multi dbs query
	{
		multiResult, err := multiquery.MultiBiSearch(ctx, req.Index*req.Limit, req.Limit, countUtils, pipe,
			countAgg, req, "ActorMessage", multiquery.ActorTransferStates)
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
