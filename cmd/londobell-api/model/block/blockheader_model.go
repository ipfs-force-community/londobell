package block

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type BlkHeaderRes struct {
	Epoch         int64
	Miner         string
	Messages      string
	ElectionProof interface{}
	Ticket        interface{}
	MessageCount  int64
}

func NewMockBlockHeaderRes() BlkHeaderRes {
	return BlkHeaderRes{
		Epoch:         123,
		Miner:         "123",
		Messages:      "123",
		ElectionProof: nil,
		Ticket:        nil,
		MessageCount:  123,
	}
}

func (m BlkHeaderRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
