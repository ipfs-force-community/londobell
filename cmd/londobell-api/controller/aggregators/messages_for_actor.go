package aggregators

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

// ctx.Addr 使用robust & ID
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

	// avoid large query scope
	latestEpoch := common.GetCurEpoch()
	if req.EndEpoch > int64(latestEpoch) {
		req.EndEpoch = int64(latestEpoch)
	}

	var (
		messagesForActorRess []model.MessageForActor
		ewg                  multierror.Group
		mutex                sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var messagesForActorRes []model.MessageForActor
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch, Addr: req.Addr}, string(messagesForActorAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &messagesForActorRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			messagesForActorRess = append(messagesForActorRess, messagesForActorRes...)
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
	sort.Slice(messagesForActorRess, func(i, j int) bool {
		return messagesForActorRess[i].Epoch > messagesForActorRess[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(messagesForActorRess) > 0 {
		tmpStartEpoch = int64(messagesForActorRess[0].Epoch) + 1
	}

	var tmpMessagesForActorRess []model.MessageForActor
	for epoch := tmpStartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var tmpMessagesForActorRes []model.MessageForActor
			tmpPipe, err := Parse(model.Ctx{StartEpoch: curEpoch, Addr: req.Addr}, string(messagesForActorAggregator))
			if err != nil {
				return err
			}

			tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
			if err != nil {
				return err
			}

			err = tmpCur.All(ctx, &tmpMessagesForActorRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			tmpMessagesForActorRess = append(tmpMessagesForActorRess, tmpMessagesForActorRes...)
			mutex.Unlock()
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	messagesForActorRess = append(messagesForActorRess, tmpMessagesForActorRess...)

	// sort
	sort.Slice(messagesForActorRess, func(i, j int) bool {
		return messagesForActorRess[i].Epoch > messagesForActorRess[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.MessagesForActorRes{TotalCount: int64(len(messagesForActorRess)), MessagesForActor: messagesForActorRess}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(messagesForActorRess)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(messagesForActorRess)) {
		res.Data = model.MessagesForActorRes{TotalCount: int64(len(messagesForActorRess[req.Index*req.Limit:])), MessagesForActor: messagesForActorRess[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.MessagesForActorRes{TotalCount: int64(len(messagesForActorRess[req.Index*req.Limit : (req.Index+1)*req.Limit])), MessagesForActor: messagesForActorRess[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
