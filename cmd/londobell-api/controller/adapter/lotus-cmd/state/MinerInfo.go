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

func GetStateMinerInfo(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateMinerInfo")
	req := lotusCmdModel.StateMinerInfoReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.Miner == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify miner to get information for"))
		return
	}

	miner, err := address.NewFromString(req.Miner)
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

	mi, err := api.StateMinerInfo(ctx, miner, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	availableBalance, err := api.StateMinerAvailableBalance(ctx, miner, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	pow, err := api.StateMinerPower(ctx, miner, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cd, err := api.StateMinerProvingDeadline(ctx, miner, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateMinerInfoRes{
		Miner:            miner,
		Epoch:            ts.Height(),
		MinerInfo:        mi,
		AvailableBalance: availableBalance,
		MinerPower:       pow,
		DeadlineInfo:     cd,
	}

	c.JSON(http.StatusOK, res)
}
