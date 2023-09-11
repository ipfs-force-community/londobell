package block

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type TipSetRes struct {
	Epoch           int64
	Cids            []string
	Weight          string
	StateRoot       string
	MessageReceipts string
	BaseFee         string
	MinTimestamp    int64
}

type TipSetListRes struct {
	Tipsets []TipSetRes
}

func NewMockTipSetRes() TipSetRes {
	return TipSetRes{
		Epoch:           123,
		Cids:            []string{"1", "2", "3"},
		Weight:          "123",
		StateRoot:       "123",
		MessageReceipts: "123",
		BaseFee:         "123",
		MinTimestamp:    123,
	}
}

func NewMockTipSetListRes() TipSetListRes {
	return TipSetListRes{
		[]TipSetRes{
			{
				Epoch:           123,
				Cids:            []string{"1", "2", "3"},
				Weight:          "123",
				StateRoot:       "123",
				MessageReceipts: "123",
				BaseFee:         "123",
				MinTimestamp:    123,
			},
			{
				Epoch:           234,
				Cids:            []string{"4", "5", "6"},
				Weight:          "234",
				StateRoot:       "234",
				MessageReceipts: "234",
				BaseFee:         "234",
				MinTimestamp:    234,
			},
		},
	}
}

func (m TipSetRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}

func (m TipSetListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
