package account

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type MessageRes struct {
	Epoch        int64
	Cid          string
	From         string
	To           string
	Nonce        uint64
	Version      int64
	Value        string
	MethodName   string
	Method       int64
	GasLimit     int64
	GasFeeCap    string
	GasPremium   string
	Params       interface{}
	Return       interface{}
	ExitCode     int64
	EventsRoot   string
	GasCost      model.GasCost
	IsBlock      bool
	SubCallCount int64
	Depth        int64
}

type MessageListRes struct {
	Messages []MessageRes
}

func NewMockMessageRes() MessageRes {
	return MessageRes{
		Epoch:      123,
		Cid:        "123",
		From:       "123",
		To:         "123",
		Nonce:      123,
		Version:    123,
		GasLimit:   123,
		GasFeeCap:  "123",
		GasPremium: "123",
		Value:      "123",
		ExitCode:   123,
		MethodName: "123",
		Method:     123,
		GasCost: model.GasCost{
			Message:            "123",
			GasUsed:            "123",
			BaseFeeBurn:        "123",
			OverEstimationBurn: "123",
			MinerPenalty:       "123",
			MinerTip:           "123",
			Refund:             "123",
			TotalCost:          "123"},
		Params:       nil,
		Return:       nil,
		IsBlock:      true,
		SubCallCount: 123,
		Depth:        123,
	}
}

func NewMockMessageListRes() MessageListRes {
	return MessageListRes{
		[]MessageRes{
			{
				Epoch:      123,
				Cid:        "123",
				From:       "123",
				To:         "123",
				Nonce:      123,
				Version:    123,
				GasLimit:   123,
				GasFeeCap:  "123",
				GasPremium: "123",
				Value:      "123",
				ExitCode:   123,
				MethodName: "123",
				Method:     123,
				GasCost: model.GasCost{
					Message:            "123",
					GasUsed:            "123",
					BaseFeeBurn:        "123",
					OverEstimationBurn: "123",
					MinerPenalty:       "123",
					MinerTip:           "123",
					Refund:             "123",
					TotalCost:          "123"},
				Params: nil,
				Return: nil,
			},
			{
				Epoch:      234,
				Cid:        "234",
				From:       "234",
				To:         "234",
				Nonce:      234,
				Version:    234,
				GasLimit:   234,
				GasFeeCap:  "234",
				GasPremium: "234",
				Value:      "234",
				ExitCode:   234,
				MethodName: "234",
				Method:     234,
				GasCost: model.GasCost{
					Message:            "234",
					GasUsed:            "234",
					BaseFeeBurn:        "234",
					OverEstimationBurn: "234",
					MinerPenalty:       "234",
					MinerTip:           "234",
					Refund:             "123",
					TotalCost:          "234"},
				Params:       nil,
				Return:       nil,
				IsBlock:      true,
				SubCallCount: 123,
				Depth:        123,
			},
		},
	}
}

func (m MessageRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}

func (m MessageListRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
