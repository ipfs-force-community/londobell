package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetTransactionByCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransactionByCid")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	traceForMessageRes, err := GetTraceByCid(ctx, req.Cid)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if len(traceForMessageRes) != 1 {
		alog.Warnf("invalid length of traceForMessageRes: %v", len(traceForMessageRes))
		c.JSON(http.StatusOK, res)
		return
	}

	trace := traceForMessageRes[0]

	msg, err := GetMessageByTrace(trace)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	tx, err := EthTxFromSignedEthMessage(msg)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.TransactionByCidRes{EthTransaction: tx}

	c.JSON(http.StatusOK, res)
}
