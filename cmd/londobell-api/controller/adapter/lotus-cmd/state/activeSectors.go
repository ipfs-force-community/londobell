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

func GetStateActiveSectors(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateActiveSectors")
	req := lotusCmdModel.StateActiveSectorsReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if req.Miner == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify miner to list sectors for"))
		return
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	activeSectors, err := api.StateMinerActiveSectors(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateActiveSectorsRes{
		Miner:         maddr,
		Epoch:         ts.Height(),
		ActiveSectors: activeSectors,
	}

	c.JSON(http.StatusOK, res)
}
