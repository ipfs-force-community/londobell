/*
资金穿透接口,根据epoch范围返回MethodName在
["InvokeContract", "Send(placeholder)", "Send", "Send(ethaccount)"]中的交易信息
*/
package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetMessagesForFund(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	alog := log.With("method", "GetMessagesForFund")
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
	transferMatchJS := `
	[
		{
			$match: {
				"ExitCode": 0,
				"MethodName": {$in: ["InvokeContract", "Send(placeholder)", "Send", "Send(ethaccount)"]},
				"Epoch": {$gte: ctx.StartEpoch,$lt: ctx.EndEpoch},
				"TransferType": {$ne:"Burn"},
				"Value": {$ne:"0"},
				"Type": "from"
			}
		},
		{
			$sort: {
				"Epoch": -1
			}
		},
		{
			$skip: ctx.Skip
		},
		{
			$limit:ctx.Limit
		},
		{
			$project: {
				_id: 0,
				Cid: {
					$cond: {
						if: {
							$eq: ["$SignedCid", null]
						}, then: "$Cid",
						else: "$SignedCid"
					}
				},
				RootCid: {
					$cond: {
						if: {
							$eq: ["$RootSignedCid", null]
						}, then: "$RootCid",
						else: "$RootSignedCid"
					}
				},            
				Epoch: "$Epoch",
				From: "$From",
				To: "$To",
				Value: "$Value",
				Method: "$MethodName",
				Depth: {
					$cond: {
						if:{
							$eq:["$IsBlock", true]
						}, then: 1,
						else: 2
					}
				},
				IsBlock: "$IsBlock"				
			}
		}
	]
`
	pipe := []byte(transferMatchJS)

	var transferMessages []model.TransferMessage

	multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, pipe, req, "ActorMessage")
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
	res.Data = model.TransferMessagesRes{TotalCount: int64(len(transferMessages)), TransferMessages: transferMessages}
	c.JSON(http.StatusOK, res)
}
