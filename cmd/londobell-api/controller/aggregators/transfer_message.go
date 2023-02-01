package aggregators

import (
	"net/http"
	"sort"
	"sync"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/hashicorp/go-multierror"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

// ctx.Addr 使用robust & ID
func GetTransferMessages(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransferMessages")
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
		transferMessagesRess []model.TransferMessage
		ewg                  multierror.Group
		mutex                sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var transferMessagesRes []model.TransferMessage
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch, Addr: req.Addr}, string(transferMessagesAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &transferMessagesRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			transferMessagesRess = append(transferMessagesRess, transferMessagesRes...)
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
	sort.Slice(transferMessagesRess, func(i, j int) bool {
		return transferMessagesRess[i].Epoch > transferMessagesRess[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(transferMessagesRess) > 0 {
		tmpStartEpoch = int64(transferMessagesRess[0].Epoch) + 1
	}

	var tmpTransferMessagesRess []model.TransferMessage
	for epoch := tmpStartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var tmpTransferMessagesRes []model.TransferMessage
			tmpPipe, err := Parse(model.Ctx{StartEpoch: curEpoch, Addr: req.Addr}, string(transferMessagesAggregator))
			if err != nil {
				return err
			}

			tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
			if err != nil {
				return err
			}

			err = tmpCur.All(ctx, &tmpTransferMessagesRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			tmpTransferMessagesRess = append(tmpTransferMessagesRess, tmpTransferMessagesRes...)
			mutex.Unlock()
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	transferMessagesRess = append(transferMessagesRess, tmpTransferMessagesRess...)

	// sort
	sort.Slice(transferMessagesRess, func(i, j int) bool {
		return transferMessagesRess[i].Epoch > transferMessagesRess[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.TransferMessagesRes{TotalCount: int64(len(transferMessagesRess)), TransferMessages: transferMessagesRess}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(transferMessagesRess)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(transferMessagesRess)) {
		res.Data = model.TransferMessagesRes{TotalCount: int64(len(transferMessagesRess[req.Index*req.Limit:])), TransferMessages: transferMessagesRess[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.TransferMessagesRes{TotalCount: int64(len(transferMessagesRess[req.Index*req.Limit : (req.Index+1)*req.Limit])), TransferMessages: transferMessagesRess[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
