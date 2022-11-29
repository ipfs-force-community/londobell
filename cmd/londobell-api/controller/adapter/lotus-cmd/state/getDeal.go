package state

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateGetDeal(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateGetDeal")
	req := lotusCmdModel.StateGetDealReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.DealID == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify deal ID"))
		return
	}

	dealid, err := strconv.ParseUint(req.DealID, 10, 64)
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

	deal, err := api.StateMarketStorageDeal(ctx, abi.DealID(dealid), ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	// todo: return value
	res.Data = lotusCmdModel.StateGetDealRes{
		Epoch:  ts.Height(),
		DealID: abi.DealID(dealid),
		Deal:   deal,
	}

	c.JSON(http.StatusOK, res)
}
