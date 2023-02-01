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

func GetTransferMessageForLargeAmount(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransferMessageForLargeAmount")
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
		transferMessageForLargeAmountRess []model.TransferMessageForLargeAmount
		ewg                               multierror.Group
		mutex                             sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var transferMessageForLargeAmountRes []model.TransferMessageForLargeAmount
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch}, string(transferMessageForLargeAmountAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &transferMessageForLargeAmountRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			transferMessageForLargeAmountRess = append(transferMessageForLargeAmountRess, transferMessageForLargeAmountRes...)
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
	sort.Slice(transferMessageForLargeAmountRess, func(i, j int) bool {
		return transferMessageForLargeAmountRess[i].Epoch > transferMessageForLargeAmountRess[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(transferMessageForLargeAmountRess) > 0 {
		tmpStartEpoch = int64(transferMessageForLargeAmountRess[0].Epoch) + 1
	}

	var tmpTransferMessageForLargeAmountRess []model.TransferMessageForLargeAmount
	for epoch := tmpStartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var tmpTransferMessageForLargeAmountRes []model.TransferMessageForLargeAmount
			tmpPipe, err := Parse(model.Ctx{StartEpoch: curEpoch}, string(transferMessageForLargeAmountAggregator))
			if err != nil {
				return err
			}

			tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
			if err != nil {
				return err
			}

			err = tmpCur.All(ctx, &tmpTransferMessageForLargeAmountRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			tmpTransferMessageForLargeAmountRess = append(tmpTransferMessageForLargeAmountRess, tmpTransferMessageForLargeAmountRes...)
			mutex.Unlock()
			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	transferMessageForLargeAmountRess = append(transferMessageForLargeAmountRess, tmpTransferMessageForLargeAmountRess...)

	// sort
	sort.Slice(transferMessageForLargeAmountRess, func(i, j int) bool {
		return transferMessageForLargeAmountRess[i].Epoch > transferMessageForLargeAmountRess[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.TransferMessagesForLargeAmountRes{TotalCount: int64(len(transferMessageForLargeAmountRess)), TransferMessagesForLargeAmount: transferMessageForLargeAmountRess}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(transferMessageForLargeAmountRess)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(transferMessageForLargeAmountRess)) {
		res.Data = model.TransferMessagesForLargeAmountRes{TotalCount: int64(len(transferMessageForLargeAmountRess[req.Index*req.Limit:])), TransferMessagesForLargeAmount: transferMessageForLargeAmountRess[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.TransferMessagesForLargeAmountRes{TotalCount: int64(len(transferMessageForLargeAmountRess[req.Index*req.Limit : (req.Index+1)*req.Limit])), TransferMessagesForLargeAmount: transferMessageForLargeAmountRess[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
