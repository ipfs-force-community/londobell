package chain

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetChainList(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainList")
	req := lotusCmdModel.ChainListReq{}
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

	if req.Count < 1 {
		util.ReturnOnErr(c, alog, fmt.Errorf("count should be set to greater than 1"))
		return
	}

	tss := make([]*types.TipSet, 0, req.Count)
	tss = append(tss, ts)

	for i := 1; i < req.Count; i++ {
		if ts.Height() == 0 {
			break
		}

		ts, err = api.ChainGetTipSet(ctx, ts.Parents())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		tss = append(tss, ts)
	}

	res.Data = lotusCmdModel.ChainListRes{
		Tipsets: tss,
	}

	c.JSON(http.StatusOK, res)
}
