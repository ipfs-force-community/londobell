package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateMarketBalance(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateMarketBalance")
	req := lotusCmdModel.StateMarketBalanceReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.Addr == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify address to print market balance for"))
		return
	}

	addr, err := address.NewFromString(req.Addr)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var ts *types.TipSet
	api := adapter.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	balance, err := api.StateMarketBalance(ctx, addr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateMarketBalanceRes{
		Addr:          addr,
		Epoch:         ts.Height(),
		MarketBalance: balance,
	}

	c.JSON(http.StatusOK, res)
}
