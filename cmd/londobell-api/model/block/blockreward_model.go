package block

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type BlkRewardRes struct {
	Miner       string
	BlockReward string
}

func NewMockBlockRewardRes() BlkRewardRes {
	return BlkRewardRes{
		Miner:       "123",
		BlockReward: "123",
	}
}

func (m BlkRewardRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
