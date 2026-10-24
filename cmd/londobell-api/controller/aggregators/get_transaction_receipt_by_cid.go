package aggregators

import (
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	cid2 "github.com/ipfs/go-cid"

	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetTransactionReceiptByCid(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
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

	traceForMessageRes, err := common.GetTraceByCid(ctx, req.Cid)
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

	msg, err := common.GetMessageByTrace(trace)
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

	txIndex, err := common.GetTransactionIndexBySeq(trace.Seq)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	api := fullnode.API.GetAppropriateAPI()

	tx, err := common.NewEthTxFromMessageLookup(ctx, trace.Epoch, msg, cid, txIndex, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 事件日志（EventsRoot 文档）缺失时**绝不能**回「空日志回执」：那与「这条消息真的没有日志」
	// 不可区分，消费方（filscan 同步器）会把「还没落库」当成「无事件」而静默跳过（2026-10-17 主网
	// fevm.erc_20_transfers 约四成高度整段缺行的根因）。
	//
	// 判据见 events_root_readiness.go：正式库提交水位已覆盖该高度仍查不到 ⇒ 留 ERROR 痕迹并保持原行为；
	// 高于正式水位（批次还没提交）或水位不可得 ⇒ 回 code:2（数据不存在＝未就绪，调用方可重试）。
	var events []types.Event
	if trace.EventsRoot != "" {
		events, err = common.GetEventsByRoot(ctx, trace.EventsRoot)
		if err != nil {
			if err != util.ErrNotFound {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			finalHeight, ferr := formalFinalHeight(ctx)
			if eventsRootNotReady(trace.Epoch, finalHeight, ferr) {
				alog.Warnw("events root not ready", "cid", req.Cid, "epoch", trace.Epoch,
					"eventsRoot", trace.EventsRoot, "finalHeight", finalHeight, "finalHeightErr", ferr)
				util.ReturnOnErr(c, util.ErrNotFound)
				return
			}

			// 已过正式提交水位却仍缺事件文档：按「事件先落库、水位后推进」的次序本不该发生。
			// 这里不谎报「无日志」（否则又是一次静默丢数），改为留 ERROR 痕迹后按空日志返回，
			// 由「关键表新鲜度/对拍」类告警去暴露真实缺口。
			alog.Errorf("events root missing below final height (suspected real data gap): cid=%s epoch=%s eventsRoot=%s finalHeight=%s",
				req.Cid, trace.Epoch, trace.EventsRoot, finalHeight)
		}
	}

	receipt, err := common.NewEthTxReceipt(ctx, tx, trace, events, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = model.TransactionReceiptByCidRes{EthReceipt: receipt}

	c.JSON(http.StatusOK, res)
}
