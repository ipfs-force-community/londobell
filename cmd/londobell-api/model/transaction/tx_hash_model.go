package transaction

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type TxHashRes struct {
	Hash string
}

func NewMockTxHashRes() TxHashRes {
	return TxHashRes{
		Hash: "123",
	}
}

func (m TxHashRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
