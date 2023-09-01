package aggregators

import (
	"encoding/json"
	"net/http"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"

	rmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetActorMessagesByMethodName(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	alog := log.With("method", "GetActorMessagesByMethodName")

	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	var count int64
	var messagesByMethodName []model.MessageByMethodName
	var createMessages []model.MessageByMethodName
	curEpoch := common.GetCurEpoch()

	req.Addr, err = GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if slices.Contains(rmodel.CreateMethods, req.MethodName) {
		creatCtx := context.WithValue(ctx, multiquery.TableKey, CreateMessageCol)
		count, createMessages, err = getActorMsgsByMethodName(creatCtx, req.Limit, count, req, curEpoch)

		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
		// update limit
		if req.Limit < int64(len(messagesByMethodName)) {
			res.Data = model.MessagesByMethodNameRes{TotalCount: count, MessagesByMethodName: createMessages[:req.Limit]}
			c.JSON(http.StatusOK, res)
			return
		}
		req.Limit = req.Limit - int64(len(messagesByMethodName))
	}

	actorCtx := context.WithValue(ctx, multiquery.TableKey, ActorMessageCol)
	count, messagesByMethodName, err = getActorMsgsByMethodName(actorCtx, req.Limit, count, req, curEpoch)

	res.Data = model.MessagesByMethodNameRes{TotalCount: count, MessagesByMethodName: append(createMessages, messagesByMethodName...)}
	c.JSON(http.StatusOK, res)
}

func getActorMsgsByMethodName(ctx context.Context, limit, count int64, req model.CommonReq, curEpoch abi.ChainEpoch) (int64, []model.MessageByMethodName, error) {
	var (
		messagesByMethodName []model.MessageByMethodName
		err                  error
		countUtils           []multiquery.CountUtil
	)

	countUtils, err = multiquery.GetTotalCountForActorMsgByMethodName(ctx, req.Addr, req.MethodName, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		return count, messagesByMethodName, err
	}

	for _, countUtil := range countUtils {
		count += countUtil.ActorMethodStates
	}

	multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.ActorMethodStates, countUtils, actorMessagesByMethodNameAggregator, req, ctx.Value(multiquery.TableKey).(string))
	if err != nil {
		return count, messagesByMethodName, err
	}

	if len(multiResult) == 0 {
		return count, messagesByMethodName, err
	}

	raw := multiResult
	rawByte, err := json.Marshal(raw)
	if err != nil {
		return count, messagesByMethodName, err
	}

	err = json.Unmarshal(rawByte, &messagesByMethodName)
	if err != nil {
		return count, messagesByMethodName, err
	}
	return count, messagesByMethodName, err

}
