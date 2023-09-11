package account

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type BalanceRes struct {
	Epoch   int64
	Balance string
}

type BalanceListRes struct {
	Balances []BalanceRes // todo:`json:",inline"`
}

func NewMockBalanceRes() BalanceRes {
	return BalanceRes{
		Epoch:   123,
		Balance: "123",
	}
}

func NewMockBalanceListRes() BalanceListRes {
	return BalanceListRes{
		[]BalanceRes{
			{
				Epoch:   123,
				Balance: "123",
			},
			{
				Epoch:   234,
				Balance: "234",
			},
		},
	}
}

func (m BalanceRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}

func (m BalanceListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
