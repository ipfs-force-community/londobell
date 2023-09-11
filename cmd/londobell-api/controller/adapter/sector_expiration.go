package adapter

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetSectorExpiration(c *gin.Context) {
	alog := log.With("method", "GetSectorExpiration")
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

	addr, err := address.NewFromString(req.ActorID)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// sector expiration
	sectors, err := api.StateMinerSectors(ctx, addr, nil, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 前65位是初始化代码，不存储
	res.Data = sectors
	c.JSON(http.StatusOK, res)
}
