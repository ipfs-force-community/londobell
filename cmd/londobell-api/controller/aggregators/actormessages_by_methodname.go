package aggregators

import (
	"encoding/json"
	"net/http"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"golang.org/x/exp/slices"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	rmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"
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
	var totalCount int64
	var messagesByMethodName []model.MessageByMethodName
	var createMessages []model.MessageByMethodName
	limit := req.Limit
	indexReq := req.Limit * req.Index
	curEpoch := common.GetCurEpoch()

	req.Addr, err = GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// CreateMiner, CreateExternal, Exec, ConstructorMethod只差create msg表
	if slices.Contains(rmodel.CreateMethods, req.MethodName) && req.MethodName != rmodel.Exec {
		creatCtx := context.WithValue(ctx, multiquery.TableKey, CreateMessageCol)
		totalCount, createMessages, err = getActorMsgsByMethodName(creatCtx, indexReq, limit, totalCount, req, curEpoch)

		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
		res.Data = model.MessagesByMethodNameRes{TotalCount: totalCount, MessagesByMethodName: createMessages}
		c.JSON(http.StatusOK, res)
		return
	}

	// 倒序,先从 actor col查询
	actorCtx := context.WithValue(ctx, multiquery.TableKey, ActorMessageCol)
	totalCount, messagesByMethodName, err = getActorMsgsByMethodName(actorCtx, indexReq, req.Limit, totalCount, req, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if req.Limit <= int64(len(messagesByMethodName)) {
		res.Data = model.MessagesByMethodNameRes{TotalCount: totalCount, MessagesByMethodName: messagesByMethodName[:req.Limit]}
		c.JSON(http.StatusOK, res)
		return
	}

	// update limit && reqIndex
	limit, indexReq = updateStartLimit(indexReq, limit, int64(len(messagesByMethodName)), totalCount)

	// search in create msg
	if req.MethodName == rmodel.Exec {
		creatCtx := context.WithValue(ctx, multiquery.TableKey, CreateMessageCol)
		totalCount, createMessages, err = getActorMsgsByMethodName(creatCtx, indexReq, limit, totalCount, req, curEpoch)

		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

	}

	res.Data = model.MessagesByMethodNameRes{TotalCount: totalCount, MessagesByMethodName: append(messagesByMethodName, createMessages...)}
	c.JSON(http.StatusOK, res)
}

func getActorMsgsByMethodName(ctx context.Context, indexReq, limit, count int64, req model.CommonReq, curEpoch abi.ChainEpoch) (int64, []model.MessageByMethodName, error) {
	var (
		messagesByMethodName []model.MessageByMethodName
		err                  error
		countUtils           []multiquery.CountUtil
		multiResult          []primitive.M
		pipe                 interface{}
	)

	countUtils, err = multiquery.GetTotalCountForActorMsgByMethodName(ctx, req.Addr, req.MethodName, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		return count, messagesByMethodName, err
	}

	for _, countUtil := range countUtils {
		count += countUtil.ActorMethodStates
	}
	// 检测越界(主要防止双表越界)
	if count <= indexReq {
		return count, messagesByMethodName, err
	}
	colName := ctx.Value(multiquery.TableKey).(string)
	if colName == CreateMessageCol {
		pipe, err = util.Parse(model.Ctx{Addr: req.Addr, MethodName: req.MethodName, Limit: req.Limit, Skip: indexReq}, monitor.GetCreateMessagesByMethodNameAggregator())
		if err != nil {
			return count, messagesByMethodName, err
		}
		multiResult, err = multiquery.MultiTraversalQuery(ctx, pipe, countUtils, colName)
	} else {

		multiResult, err = multiquery.MultiBiSearch(ctx, indexReq, req.Limit, countUtils, monitor.GetActorMessagesByMethodNameNoskipAggregator(),
			monitor.GetCountOfActorMessagesByMethodNameAggregator(), req, colName, multiquery.ActorMethodStates)
	}

	// multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.ActorMethodStates, countUtils, actorMessagesByMethodNameAggregator, req, ctx.Value(multiquery.TableKey).(string))
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
