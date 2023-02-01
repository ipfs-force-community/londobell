package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

func ReturnOnErr(c *gin.Context, err error) {
	if err != nil {
		res := model.CommonRes{}
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res) // todo: status code
	}
}
