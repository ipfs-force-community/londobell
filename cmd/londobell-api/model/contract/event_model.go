package contract

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type EventRes struct {
	ActorID  string
	Epoch    int64
	Cid      string
	Topics   []string
	Data     string
	LogIndex int64
	Removed  bool
}

type EventListRes struct {
	Events []EventRes
}

func NewMockEventListRes() EventListRes {
	return EventListRes{
		[]EventRes{
			{
				ActorID:  "123",
				Epoch:    123,
				Cid:      "123",
				Topics:   []string{"1", "2", "3"},
				Data:     "123",
				LogIndex: 123,
				Removed:  false,
			},
			{
				ActorID:  "234",
				Epoch:    234,
				Cid:      "234",
				Topics:   []string{"2", "3", "4"},
				Data:     "234",
				LogIndex: 234,
				Removed:  false,
			},
		},
	}
}

func (m EventListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
