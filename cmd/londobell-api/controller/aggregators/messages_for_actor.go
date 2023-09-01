package aggregators

import (
	"encoding/json"
	"math"
	"net/http"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

var (
	CreateMessageCol = "CreateMessage"
	ActorMessageCol  = "ActorMessage"
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

	actorID, err := GetIDByAddr(ctx, req.Addr)
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

	// first search In create col
	creatCtx := context.WithValue(ctx, multiquery.TableKey, CreateMessageCol)
	messagesForCreate, totalCount, err = getActorMsgs(creatCtx, req.Limit, totalCount, req, curEpoch)

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// update limit
	if req.Limit < int64(len(messagesForCreate)) {
		res.Data = model.MessagesForActorRes{TotalCount: req.Limit, MessagesForActor: messagesForCreate[:req.Limit]}
		c.JSON(http.StatusOK, res)
		return
	}
	limit := req.Limit - int64(len(messagesForCreate))

	// search actor col
	actorCtx := context.WithValue(ctx, multiquery.TableKey, ActorMessageCol)
	messagesForActor, totalCount, err = getActorMsgs(actorCtx, limit, totalCount, req, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: append(messagesForCreate, messagesForActor...)}
	c.JSON(http.StatusOK, res)
}

func getActorMsgs(ctx context.Context, limit, count int64, req model.CommonReq, curEpoch abi.ChainEpoch) ([]model.MessageForActor, int64, error) {
	var (
		messagesForActor []model.MessageForActor
		countUtils       []multiquery.CountUtil
		err              error
	)
	colName := ctx.Value(multiquery.TableKey).(string)
	countUtils, err = multiquery.GetTotalCountForActorMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		return messagesForActor, count, err
	}
	for _, countUtil := range countUtils {
		count += countUtil.ActorStates
	}

	multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, limit, multiquery.ActorStates, countUtils, messagesForActorAggregator, req, colName)
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
