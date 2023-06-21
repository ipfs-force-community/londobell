package adapter

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/evm"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetInitCodeForEvm(c *gin.Context) {
	alog := log.With("method", "GetInitCodeForEvm")
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

	ts, err := api.ChainHead(ctx)
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

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

	act, err := api.StateGetActor(ctx, addr, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if !builtin.IsEvmActor(act.Code) {
		err = fmt.Errorf("not evm actor for: %v", act.Code)
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	st, err := evm.Load(stor, act)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	byteCode, err := st.GetBytecode()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 前65位是初始化代码，不存储
	res.Data = model.InitCodeForEvmRes{InitCode: hex.EncodeToString(byteCode)}
	c.JSON(http.StatusOK, res)
}
