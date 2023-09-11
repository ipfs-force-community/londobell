package block

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type BlkCountRes struct {
	Epoch        int64 `json:"_id" bson:"_id"`
	MessageCount int64
	BlockCount   int64
	BaseFee      string
}

type BlkCountListRes struct {
	BlockCounts []BlkCountRes
}

func NewMockBlockCountListRes() BlkCountListRes {
	return BlkCountListRes{
		[]BlkCountRes{
			{
				Epoch:        123,
				MessageCount: 123,
				BlockCount:   123,
				BaseFee:      "123",
			},
			{
				Epoch:        234,
				MessageCount: 234,
				BlockCount:   234,
				BaseFee:      "234",
			},
		},
	}
}

func (m BlkCountListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
