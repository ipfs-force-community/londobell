package contract

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type InitCodeRes struct {
	InitCode string
}

func NewMockInitCodeRes() InitCodeRes {
	return InitCodeRes{InitCode: "123"}
}

func (m InitCodeRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
