package adapter

import (
	"net/http"

	"github.com/filecoin-project/lotus/build"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetVersion(c *gin.Context) {
	alog := log.With("method", "GetVersion")
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	
	res.Data = build.UserVersion()

	c.JSON(http.StatusOK, res)
}
