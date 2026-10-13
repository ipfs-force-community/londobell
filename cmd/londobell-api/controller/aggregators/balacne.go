package aggregators

import (
	"encoding/json"
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetBalance(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBalance")
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

	// convert to ID
	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var balanceRes []model.BalanceRes

	// multi dbs query
	{
		// ActorBalance is only written in full at the zero hour; every regular
		// tick carries a delta. Forward-fill from the most recent full snapshot
		// so actors that did not move keep reporting their carried-forward
		// balance. That snapshot bounds the scan window, which guarantees every
		// actor has a row inside it.
		fromEpoch, ok := latestActorBalanceFullSnapshotEpoch(req.StartEpoch)
		if !ok {
			alog.Warnf("no full ActorBalance snapshot at or before %d, querying that epoch alone", req.StartEpoch)
		}

		multiResult, err := multiquery.MultiRangeQuery(ctx, fromEpoch, req.StartEpoch+1, countUtils, common2.BalanceForwardFillPipeline(common2.BalanceAggregator), req, "ActorBalance")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &balanceRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		// One row per shard was returned (each sorted Epoch desc, limited to
		// 1); keep the newest. Report it under the requested epoch so the
		// response shape is unchanged and the frontend needs no change.
		if len(balanceRes) > 1 {
			newest := 0
			for i := range balanceRes {
				if balanceRes[i].Epoch > balanceRes[newest].Epoch {
					newest = i
				}
			}
			balanceRes = []model.BalanceRes{balanceRes[newest]}
		}
		for i := range balanceRes {
			balanceRes[i].Epoch = abi.ChainEpoch(req.StartEpoch)
		}
	}

	res.Data = balanceRes
	c.JSON(http.StatusOK, res)
}
