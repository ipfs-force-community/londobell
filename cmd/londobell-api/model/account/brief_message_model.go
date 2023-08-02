package account

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type BriefMessageListRes struct {
	BriefMessages []BriefMessage
}

type BriefMessage struct {
	Epoch      int64
	Cid        string
	From       string
	To         string
	Value      string
	MethodName string
	IsBlock    bool
}

func NewMockBriefMessageListRes() BriefMessageListRes {
	return BriefMessageListRes{
		[]BriefMessage{
			{
				Epoch:      123,
				Cid:        "123",
				From:       "123",
				To:         "123",
				Value:      "123",
				MethodName: "123",
				IsBlock:    true,
			},
			{
				Epoch:      234,
				Cid:        "234",
				From:       "234",
				To:         "234",
				Value:      "234",
				MethodName: "234",
				IsBlock:    false,
			},
		},
	}
}

func (m BriefMessageListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
