package aggregators

import (
	"math"
	"net/http"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetCountAndMethodsOfMessagesForBlockHeader(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetCountAndMethodsOfMessagesForBlockHeader")
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
	req.Limit = math.MaxInt64
	messagesForBlockRes, err := getBlockMsg(req, countUtils)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	var countAndMethodsForBlockHeaderRes []model.CountAndMethodsForBlockHeader

	methodCount := make(map[string]int64)
	for _, blockMsg := range messagesForBlockRes {
		methodCount[blockMsg.Method]++
	}
	for method, count := range methodCount {
		countAndMethodsForBlockHeaderRes = append(countAndMethodsForBlockHeaderRes, model.CountAndMethodsForBlockHeader{
			MethodName: method,
			Count:      count,
		})
	}
	res.Data = countAndMethodsForBlockHeaderRes
	c.JSON(http.StatusOK, res)
}
