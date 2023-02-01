package aggregators

import (
	"net/http"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	"golang.org/x/net/context"
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

	// avoid large query scope
	latestEpoch := common.GetCurEpoch()
	if req.EndEpoch > int64(latestEpoch) {
		req.EndEpoch = int64(latestEpoch)
	}

	var (
		messagesByMethodNameRess []model.MessageByMethodName
		ewg                      multierror.Group
		mutex                    sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var messagesByMethodNameRes []model.MessageByMethodName
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch, MethodName: req.MethodName, Addr: req.Addr}, string(actorMessagesByMethodNameAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &messagesByMethodNameRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			messagesByMethodNameRess = append(messagesByMethodNameRess, messagesByMethodNameRes...)
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
	sort.Slice(messagesByMethodNameRess, func(i, j int) bool {
		return messagesByMethodNameRess[i].Epoch > messagesByMethodNameRess[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(messagesByMethodNameRess) > 0 {
		tmpStartEpoch = int64(messagesByMethodNameRess[0].Epoch) + 1
	}

	var tmpMessagesByMethodNameRess []model.MessageByMethodName

	for epoch := tmpStartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var tmpMessagesByMethodNameRes []model.MessageByMethodName
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch, MethodName: req.MethodName, Addr: req.Addr}, string(actorMessagesByMethodNameAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &tmpMessagesByMethodNameRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			tmpMessagesByMethodNameRess = append(tmpMessagesByMethodNameRess, tmpMessagesByMethodNameRes...)
			mutex.Unlock()
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	messagesByMethodNameRess = append(messagesByMethodNameRess, tmpMessagesByMethodNameRess...)

	// sort
	sort.Slice(messagesByMethodNameRess, func(i, j int) bool {
		return messagesByMethodNameRess[i].Epoch > messagesByMethodNameRess[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.MessagesByMethodNameRes{TotalCount: int64(len(messagesByMethodNameRess)), MessagesByMethodName: messagesByMethodNameRess}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(messagesByMethodNameRess)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(messagesByMethodNameRess)) {
		res.Data = model.MessagesByMethodNameRes{TotalCount: int64(len(messagesByMethodNameRess[req.Index*req.Limit:])), MessagesByMethodName: messagesByMethodNameRess[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.MessagesByMethodNameRes{TotalCount: int64(len(messagesByMethodNameRess[req.Index*req.Limit : (req.Index+1)*req.Limit])), MessagesByMethodName: messagesByMethodNameRess[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
