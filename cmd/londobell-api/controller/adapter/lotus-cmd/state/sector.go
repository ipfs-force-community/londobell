package state

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateSector(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateSector")
	req := lotusCmdModel.StateSectorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.Miner == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must specify miner to get sector info"))
		return
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	sid, err := strconv.ParseInt(req.SectorNumber, 10, 64)
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

	si, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(sid), ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}
	if si == nil {
		util.ReturnOnErr(c, alog, fmt.Errorf("sector %d for miner %s not found", sid, maddr))
		return
	}

	sp, err := api.StateSectorPartition(ctx, maddr, abi.SectorNumber(sid), ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateSectorRes{
		Miner:        maddr,
		Epoch:        ts.Height(),
		SectorNumber: abi.SectorNumber(sid),
		Sector:       si,
		Partition:    sp,
	}

	c.JSON(http.StatusOK, res)
}
