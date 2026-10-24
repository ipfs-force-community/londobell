package aggregators

import (
	"context"
	"net/http"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetAllMethodsForPendingMessages(c *gin.Context) {
	alog := log.With("method", "GetAllMethodsForPendingMessages")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	unStoredMsgs, _, err := getUnStoredMsgs(ctx, api)

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 同 mpool 的默认分支：项数由 mpool 条数决定，不能每项起一个 goroutine（2026-10-24 OOM 根因）。
	// 先按固定并发取 method name，再在主协程里累加，避免为计数再引入锁。
	type poolItem struct {
		cur *types.TipSet
		msg *types.SignedMessage
	}

	items := make([]poolItem, 0, len(unStoredMsgs))
	for cur, msgs := range unStoredMsgs {
		for _, msg := range msgs {
			items = append(items, poolItem{cur: cur, msg: msg})
		}
	}

	methodNames := make([]string, len(items))
	if err := util.RunBounded(ctx, len(items), func(ctx context.Context, idx int) error {
		it := items[idx]
		methodName, err := adapter.GetMethodName(ctx, alog, api, it.msg, it.cur)
		if err != nil && err != util.ErrNotFound {
			return err
		}
		methodNames[idx] = methodName
		return nil
	}); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	allMethods := make(map[string]int64)
	for _, methodName := range methodNames {
		allMethods[methodName]++
	}

	var allMethodsRes []model.AllMethodsRes
	for methodName, count := range allMethods {
		allMethodsRes = append(allMethodsRes, model.AllMethodsRes{MethodName: methodName, Count: count})
	}

	res.Data = allMethodsRes
	c.JSON(http.StatusOK, res)
}
