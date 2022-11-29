package chain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/stmgr"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetChainDecode(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainDecode")
	req := lotusCmdModel.ChainDecodeReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.To == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass to address"))
		return
	}
	to, err := address.NewFromString(req.To)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if req.Method == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass Method number"))
		return
	}
	method, err := strconv.ParseInt(req.Method, 10, 64)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if req.Params == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass params"))
		return
	}

	var params []byte
	switch req.Encoding {
	case "base64":
		params, err = base64.StdEncoding.DecodeString(req.Params)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	case "hex":
		params, err = hex.DecodeString(req.Params)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	default:
		util.ReturnOnErr(c, alog, fmt.Errorf("unrecognized encoding: %s", req.Encoding))
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

	act, err := api.StateGetActor(ctx, to, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	p, err := stmgr.GetParamType(filcns.NewActorRegistry(), act.Code, abi.MethodNum(method)) // todo use api for correct actor registry
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if err := p.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainDecodeRes{
		Params: p,
	}

	c.JSON(http.StatusOK, res)
}
