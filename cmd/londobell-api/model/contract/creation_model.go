package contract

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type CreationRes struct {
	Epoch   int64
	Creator string
}

func NewMockCreationRes() CreationRes {
	return CreationRes{
		Epoch:   123,
		Creator: "123",
	}
}

func (m CreationRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
