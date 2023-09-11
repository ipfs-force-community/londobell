package aggregators

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

var (
	CreateMessageCol            = "CreateMessage"
	ActorMessageCol             = "ActorMessage"
	ActorMsgCountKey            = "ActorMsgCount"
	MessagesForCreateAggregator = `
	// CreateMessage
	// ms ActorID_1_IsBlock_1_Epoch_1
	[
		{
			$match: {
				"ActorID": ctx.Addr,
				"IsBlock": true,
			}
		},
		{
			$sort: {
				Epoch: -1
			}
		},
		{
			$skip: ctx.Skip
		},		
		{
			$limit: ctx.Limit
		},
		{
			$project: {
				_id: 0,
				Cid: {
					$cond: {
						if:{
							$eq:["$SignedCid", null]
						}, then: "$Cid",
						else: "$SignedCid"
					}
				},
				Epoch: "$Epoch",
				From: "$From",
				To: "$To",
				Value: "$Value",
				ExitCode: "$ExitCode",
				Method: "$MethodName",
			}
		}
	]	
	`
)

func GetMessagesForActor(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	alog := log.With("method", "GetMessagesForActor")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()
	totalCount := int64(0)

	actorID, err := common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr = actorID

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	var messagesForActor []model.MessageForActor
	var messagesForCreate []model.MessageForActor
	limit := req.Limit
	indexReq := req.Limit * req.Index

	// 倒序,先从 actor col查询
	actorCtx := context.WithValue(ctx, multiquery.TableKey, ActorMessageCol)
	messagesForActor, totalCount, err = getActorMsgs(actorCtx, indexReq, limit, totalCount, req, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if limit <= int64(len(messagesForActor)) {
		res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: messagesForActor[:req.Limit]}
		c.JSON(http.StatusOK, res)
		return
	}

	// update limit && reqIndex actor数据不足reqIndex
	limit, indexReq = updateStartLimit(indexReq, limit, int64(len(messagesForActor)), totalCount)

	// search In create col
	creatCtx := context.WithValue(ctx, multiquery.TableKey, CreateMessageCol)
	messagesForCreate, totalCount, err = getActorMsgs(creatCtx, indexReq, limit, totalCount, req, curEpoch)

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: append(messagesForActor, messagesForCreate...)}
	c.JSON(http.StatusOK, res)
}

func getActorMsgs(ctx context.Context, indexReq, limit, count int64, req model.CommonReq, curEpoch abi.ChainEpoch) ([]model.MessageForActor, int64, error) {
	var (
		messagesForActor []model.MessageForActor
		countUtils       []multiquery.CountUtil
		err              error
		multiResult      []primitive.M
		pipe             interface{}
	)
	colName := ctx.Value(multiquery.TableKey).(string)
	countUtils, err = multiquery.GetTotalCountForActorMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		return messagesForActor, count, err
	}
	for _, countUtil := range countUtils {
		count += countUtil.ActorStates
	}

	// 检测越界(主要防止双表越界)
	if count <= indexReq {
		return messagesForActor, count, err
	}

	if colName == CreateMessageCol {

		pipe, err = util.Parse(model.Ctx{Addr: req.Addr, Limit: limit, Skip: indexReq}, monitor.GetMessagesForCreateAggregator())
		if err != nil {
			return messagesForActor, count, err
		}
		multiResult, err = multiquery.MultiTraversalQuery(ctx, pipe, countUtils, colName)
	} else {
		multiResult, err = multiquery.MultiBiSearch(ctx, indexReq, limit, countUtils, common2.ActorMessageNoSkip,
			monitor.GetCountOfMessageForActorAggregator(), req, colName, multiquery.ActorStates)
	}

	// multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, limit, multiquery.ActorStates, countUtils, messagesForActorAggregator, req, colName)

	if err != nil {
		return messagesForActor, count, err
	}

	if len(multiResult) != 0 {
		raw := multiResult

		rawByte, err := json.Marshal(raw)
		if err != nil {
			return messagesForActor, count, err
		}

		err = json.Unmarshal(rawByte, &messagesForActor)

		if err != nil {
			return messagesForActor, count, err
		}
	}
	return messagesForActor, count, nil
}

// 多表查询,更新reqLimit*req.Index Limit
func updateStartLimit(start, limit, lastResultsLen, lastCount int64) (int64, int64) {

	// start,limit 为create msg结果的左右边界
	if start < lastCount {
		start = 0
	} else {
		start -= lastCount
	}

	limit -= lastResultsLen

	return limit, start
}
