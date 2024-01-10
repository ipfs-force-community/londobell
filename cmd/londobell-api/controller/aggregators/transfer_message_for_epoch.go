package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

var baseTransferPipe = `
[
	{
		$match: {
			"ActorID": ctx.Addr,
			"ExitCode": 0,
			"IsBlock": true,
			"MethodName": {$in: ["Send", "Send(placeholder)"]},
			"Epoch": {$lte: ctx.EndEpoch},
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
            Method: "$MethodName"
        }
    }
]
`

func GetTransferMessageByEpoch(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransferMessageForEpoch")
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
	)

	countUtils, err = multiquery.GetTotalCountForActorTransferMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pipe = []byte(baseTransferPipe)

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.ActorTransferStates
	}

	var transferMessages []model.TransferMessageIMToken
	// multi dbs query
	{
		// multiResult, err := multiquery.MultiBiSearch(ctx, req.Index*req.Limit, req.Limit, countUtils, pipe,
		// 	countAgg, req, "ActorMessage", multiquery.ActorTransferStates)
		multiResult, err := multiquery.MultiRangeQuery2(ctx, req.StartEpoch, req.EndEpoch, countUtils, pipe, req, "ActorMessage")
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

	res.Data = model.TransferMessagesIMTokenRes{TotalCount: totalCount, TransferMessages: transferMessages}
	c.JSON(http.StatusOK, res)
}
