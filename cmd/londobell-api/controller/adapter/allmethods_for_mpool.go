package adapter

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	ts, err := api.ChainHead(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	msgs, err := api.MpoolPending(ctx, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var (
		allMethods = make(map[string]int64)
		g          multierror.Group
		mutex      sync.Mutex
	)

	for i := range msgs {
		msg := msgs[i]
		g.Go(func() error {
			methodName, err := getMethodName(ctx, alog, api, msg, ts)
			if err != nil {
				return err
			}

			mutex.Lock()
			allMethods[methodName]++
			mutex.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var allMethodsRes []model.AllMethodsRes
	for methodName, count := range allMethods {
		allMethodsRes = append(allMethodsRes, model.AllMethodsRes{MethodName: methodName, Count: count})
	}

	res.Data = allMethodsRes
	c.JSON(http.StatusOK, res)
}
