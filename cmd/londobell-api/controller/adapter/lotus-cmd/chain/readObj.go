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

func GetChainReadObj(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainReadObj")
	req := lotusCmdModel.ChainReadObjReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if req.ObjectCid == "" {
		util.ReturnOnErr(c, alog, fmt.Errorf("must pass cid of object to print"))
		return
	}

	cid, err := cid.Decode(req.ObjectCid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	api := adapter.API.GetAppropriateAPI()

	obj, err := api.ChainReadObj(ctx, cid)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.ChainReadObjRes{
		ObjectCid: cid,
		Object:    obj,
	}

	c.JSON(http.StatusOK, res)
}
