package aggregators

import (
	"fmt"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	cid2 "github.com/ipfs/go-cid"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetTransactionReceiptByCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTransactionReceiptByCid")
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
		err = fmt.Errorf("invalid length of traceForMessageRes: %v", len(traceForMessageRes))
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	trace := traceForMessageRes[0]

	msg, err := GetMessageByTrace(trace)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cid, err := cid2.Decode(trace.Cid)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	txIndex, err := GetTransactionIndexBySeq(trace.Seq)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	api := fullnode.API.GetAppropriateAPI()

	tx, err := newEthTxFromMessageLookup(ctx, trace.Epoch, msg, cid, txIndex, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	events, err := GetEventsByRoot(ctx, trace.EventsRoot)
	if err != nil && err != ErrNotFound {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	receipt, err := newEthTxReceipt(ctx, tx, trace, events, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.TransactionReceiptByCidRes{EthReceipt: receipt}

	c.JSON(http.StatusOK, res)
}
