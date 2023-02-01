package aggregators

import (
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs-force-community/londobell/common"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetAllMethodsForActor(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAllMethodsForActor")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// avoid large query scope
	latestEpoch := common.GetCurEpoch()
	if req.EndEpoch > int64(latestEpoch) {
		req.EndEpoch = int64(latestEpoch)
	}

	var (
		allMethodsRess []model.AllMethodsRes
		ewg            multierror.Group
		mutex          sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var allMethodsRes []model.AllMethodsRes
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch, Addr: req.Addr}, string(allMethodsForActorAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.MessageCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &allMethodsRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			allMethodsRess = append(allMethodsRess, allMethodsRes...)
			mutex.Unlock()
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// get near-height data from the temporary repository

	var (
		methodMap        = make(map[string]struct{})
		allMethods       []string
		allUniqueMethods []string
	)

	for _, methods := range allMethodsRess {
		allMethods = append(allMethods, methods.AllMethods...)
	}

	for _, method := range allMethods {
		if _, ok := methodMap[method]; !ok {
			methodMap[method] = struct{}{}
			allUniqueMethods = append(allUniqueMethods, method)
		}
	}

	res.Data = model.AllMethodsRes{AllMethods: allUniqueMethods}
	c.JSON(http.StatusOK, res)
}
