package chain

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs/go-cid"
)

func GetChainStatObj(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainStatObj")
	req := lotusCmdModel.ChainStatObjReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.Cid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass cid to print"))
		return
	}

	obj, err := cid.Decode(req.Cid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	base := cid.Undef
	if req.Base != "" {
		base, err = cid.Decode(req.Base)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	}

	api := adapter.API.GetAppropriateAPI()

	stats, err := api.ChainStatObj(ctx, obj, base)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainStatObjRes{
		Cid:     obj,
		Base:    base,
		ObjStat: stats,
	}

	c.JSON(http.StatusOK, res)
}
