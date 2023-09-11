package adapter

import (
	"context"
	"net/http"

	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/market"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBalanceAtMarket(c *gin.Context) {
	alog := log.With("method", "GetBalanceAtMarket")
	req := model.ActorReq{}
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

	var ts *types.TipSet
	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	mact, err := api.StateGetActor(ctx, sbuiltin.StorageMarketActorAddr, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))
	mas, err := market.Load(stor, mact)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	escrowTable, err := mas.EscrowTable()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	lockedTable, err := mas.LockedTable()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData := make([]model.MarketBalanceRes, len(req.Addrs))
	for i, addr := range req.Addrs {
		actor, err := address.NewFromString(addr)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		escrow, err := escrowTable.Get(actor)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		locked, err := lockedTable.Get(actor)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		resData[i] = model.MarketBalanceRes{
			Actor:         actor,
			Epoch:         ts.Height(),
			EscrowBalance: escrow,
			LockedBalance: locked,
		}

	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
