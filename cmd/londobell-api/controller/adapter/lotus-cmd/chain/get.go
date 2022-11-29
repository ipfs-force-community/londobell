package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v8/account"
	"github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/go-state-types/builtin/v8/miner"
	"github.com/filecoin-project/go-state-types/builtin/v8/power"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetChainGet(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainGet")
	req := lotusCmdModel.ChainGetReq{}
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

	p := path.Clean(req.Path)
	if strings.HasPrefix(p, "/pstate") {
		p = p[len("/pstate"):]

		p = "/ipfs/" + ts.ParentState().String() + p
	}

	obj, err := api.ChainGetNode(ctx, p)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	t := strings.ToLower(req.AsType)
	if t == "" {
		b, err := json.MarshalIndent(obj.Obj, "", "\t")
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		res.Data = lotusCmdModel.ChainGetRes{
			Path:    p,
			DAGNode: b,
		}

		c.JSON(http.StatusOK, res)
		return
	}

	var cbu cbg.CBORUnmarshaler
	switch t {
	case "raw":
		cbu = nil
	case "block":
		cbu = new(types.BlockHeader)
	case "message":
		cbu = new(types.Message)
	case "smessage", "signedmessage":
		cbu = new(types.SignedMessage)
	case "actor":
		cbu = new(types.Actor)
	case "amt":
		err = handleAmt(ctx, api, obj.Cid)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	case "hamt-epoch":
		err = handleHamtEpoch(ctx, api, obj.Cid)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	case "hamt-address":
		err = handleHamtAddress(ctx, api, obj.Cid)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	case "cronevent":
		cbu = new(power.CronEvent)
	case "account-state":
		cbu = new(account.State)
	case "miner-state":
		cbu = new(miner.State)
	case "power-state":
		cbu = new(power.State)
	case "market-state":
		cbu = new(market.State)
	default:
		util.ReturnOnErr(c, alog, fmt.Errorf("unknown type: %q", t))
		return
	}

	raw, err := api.ChainReadObj(ctx, obj.Cid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if cbu == nil {
		res.Data = lotusCmdModel.ChainGetRes{
			Path:    p,
			DAGNode: raw,
		}

		c.JSON(http.StatusOK, res)
		return
	}

	if err := cbu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainGetRes{
		Path:    p,
		DAGNode: cbu,
	}

	c.JSON(http.StatusOK, res)
}
