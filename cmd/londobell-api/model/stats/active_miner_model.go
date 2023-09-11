package stats

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type ActiveMinerRes struct {
	ActiveMiners []string
}

func NewMockActiveMinerRes() ActiveMinerRes {
	return ActiveMinerRes{
		ActiveMiners: []string{"1", "2", "3"},
	}
}

func (m ActiveMinerRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
