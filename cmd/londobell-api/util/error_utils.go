package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

func ReturnOnErr(c *gin.Context, log *zap.SugaredLogger, err error) {
	if err != nil {
		log.Error(err)
		res := model.CommonRes{}
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
	}
}
