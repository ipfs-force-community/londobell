package util

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

var ErrNotFound = fmt.Errorf("get wrong result, length of result shoule be one")

func ReturnOnErr(c *gin.Context, err error) {
	if err != nil {
		res := model.CommonRes{}
		if err == ErrNotFound {
			res.Code = model.NotFound
		} else {
			res.Code = model.Fail
		}

		res.Msg = err.Error()
		c.JSON(http.StatusOK, res) // todo: status code
		return
	}
}
