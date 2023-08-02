package stats

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type NewActorRes struct {
	NewActors []string
}

func NewMockNewActorRes() NewActorRes {
	return NewActorRes{
		NewActors: []string{"1", "2", "3"},
	}
}

func (m NewActorRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
