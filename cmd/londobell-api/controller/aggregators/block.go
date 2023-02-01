package aggregators

import (
	"net/http"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs-force-community/londobell/common"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetBlock(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBlock")
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
		ewg       multierror.Group
		blockRess []model.BlockMessage
		mutex     sync.Mutex
	)

	for epoch := req.StartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var blockRes []model.BlockMessage
			pipe, err := Parse(model.Ctx{StartEpoch: curEpoch}, string(blockAggregator))
			if err != nil {
				return err
			}

			cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
			if err != nil {
				return err
			}

			err = cur.All(ctx, &blockRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			blockRess = append(blockRess, blockRes...)
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
	sort.Slice(blockRess, func(i, j int) bool {
		return blockRess[i].Epoch > blockRess[j].Epoch
	})

	tmpStartEpoch := req.StartEpoch
	if len(blockRess) > 0 {
		tmpStartEpoch = int64(blockRess[0].Epoch) + 1
	}

	var tmpBlockRess []model.BlockMessage
	for epoch := tmpStartEpoch; epoch < req.EndEpoch; epoch++ {
		curEpoch := epoch
		ewg.Go(func() error {
			var tmpBlockRes []model.BlockMessage
			tmpPipe, err := Parse(model.Ctx{StartEpoch: curEpoch}, string(blockAggregator))
			if err != nil {
				return err
			}

			tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
			if err != nil {
				return err
			}

			err = tmpCur.All(ctx, &tmpBlockRes)
			if err != nil {
				return err
			}

			mutex.Lock()
			tmpBlockRess = append(tmpBlockRess, tmpBlockRes...)
			mutex.Unlock()

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	blockRess = append(blockRess, tmpBlockRess...)

	// sort
	sort.Slice(blockRess, func(i, j int) bool {
		return blockRess[i].Epoch > blockRess[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.BlockMessagesRes{TotalCount: int64(len(blockRess)), BlockMessages: blockRess}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(blockRess)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(blockRess)) {
		res.Data = model.BlockMessagesRes{TotalCount: int64(len(blockRess[req.Index*req.Limit:])), BlockMessages: blockRess[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.BlockMessagesRes{TotalCount: int64(len(blockRess[req.Index*req.Limit : (req.Index+1)*req.Limit])), BlockMessages: blockRess[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}
