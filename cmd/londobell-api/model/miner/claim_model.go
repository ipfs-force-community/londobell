package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type ClaimRes struct {
	Claims []Claim
}

type Claim struct {
	Epoch     int64
	Miner     string
	Client    string
	Sector    int64
	ClaimID   int64
	Size      uint64
	Data      string
	TermStart int64
	TermMin   int64
	TermMax   int64
}

func NewMockClaimRes() ClaimRes {
	return ClaimRes{
		[]Claim{
			{
				Epoch:     123,
				Miner:     "123",
				Client:    "123",
				Sector:    123,
				ClaimID:   123,
				Size:      123,
				Data:      "123",
				TermStart: 123,
				TermMin:   123,
				TermMax:   123,
			},
			{
				Epoch:     234,
				Miner:     "234",
				Client:    "234",
				Sector:    234,
				ClaimID:   234,
				Size:      234,
				Data:      "234",
				TermStart: 234,
				TermMin:   234,
				TermMax:   123,
			},
		},
	}
}

func (m ClaimRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
