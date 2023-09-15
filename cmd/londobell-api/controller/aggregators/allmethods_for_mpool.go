package aggregators

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	unStoredMsgs, _, err := getUnStoredMsgs(ctx, api)

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
	for cur, msgs := range unStoredMsgs {
		for i := range msgs {
			cur := cur
			msg := msgs[i]
			g.Go(func() error {
				methodName, err := adapter.GetMethodName(ctx, alog, api, msg, cur)
				if err != nil && err != util.ErrNotFound {
					return err
				}

				mutex.Lock()
				allMethods[methodName]++
				mutex.Unlock()

				return nil
			})
		}
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
