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

	// Answer only the epochs this instance has published. A request past the
	// top of that range is out of bounds and must answer data:null — never the
	// newest written balance carried forward onto an epoch that has not
	// happened. The pre-change reader got this for free from its forward-fill
	// window bound; see planBalanceQuery for why that no longer works and what
	// the answerable range is.
	plan := planBalanceQuery(req.StartEpoch, countUtils)
	if !plan.Answerable {
		alog.Debugf("balance request epoch %d outside answerable range, returning null", req.StartEpoch)
		c.JSON(http.StatusOK, res)
		return
	}

	// multi dbs query
	{
		// There is no longer any epoch carrying a full ActorBalance row-set (the
		// zero hour writes a delta like every other snapshot), so the window must
		// reach back to the beginning of the chain: an actor that last changed
		// before the latest snapshot has no row inside any narrower window, and a
		// forward-fill over a short window would silently return nothing for it.
		// The pipeline keeps its Epoch range, so this stays an index seek on
		// Epoch_1_Addr_1 rather than a collection scan.
		multiResult, err := multiquery.MultiRangeQuery(ctx, plan.FromEpoch, plan.ToEpoch, countUtils, common2.BalanceForwardFillPipeline(common2.BalanceAggregator), req, "ActorBalance")
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
