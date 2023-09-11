package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type Pledge struct {
	Epoch             int64
	PreCommitDeposits string
	LockedFunds       string
	FeeDebt           string
	InitialPledge     string
	AvailableBalance  string
}

type PledgeRes struct {
	Pledges []Pledge
}

func NewMockPledgeRes() PledgeRes {
	return PledgeRes{
		Pledges: []Pledge{
			{
				Epoch:             123,
				PreCommitDeposits: "123",
				LockedFunds:       "123",
				FeeDebt:           "123",
				InitialPledge:     "123",
				AvailableBalance:  "123",
			},
			{
				Epoch:             234,
				PreCommitDeposits: "234",
				LockedFunds:       "234",
				FeeDebt:           "234",
				InitialPledge:     "234",
				AvailableBalance:  "234",
			},
		},
	}
}

func (m PledgeRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
