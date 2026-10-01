package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

// 分页参数归一化（2026-10-01 事故修复，见 /tipset/chain 页面 OOM）。
//
// 历史行为：limit<=0 会被 multi-query.MultiPagingQuery 解释成「取全量」
//（其内部把 limit 置为 math.MaxInt64；query.go 里还留着 "todo: skip=0 & limit=0 获取全量？"）。
// 在高链长的库上（cali 实测 bell.Tipset 2,434,366 条），一次 limit=0 就会把整段全量物化：
// 实测单请求把进程 RSS 顶到 15.4GB（dmesg anon-rss:15402712kB）并触发内核 OOM，
// 聚合器进入重启循环，页面从「慢」变成 500/超时。
//
// 列表接口永远不需要「全部」，所以这里把 <=0 归一成默认页大小；正数原样透传
//（真正过大的单次结果由 multi-query/common.BoundedAll 的字节上限显式报错兜底，
//  那是「显式失败」而不是「悄悄吃掉 15GB」）。
const defaultTipsetsPageLimit int64 = 20

func normalizeTipsetsLimit(limit int64) int64 {
	if limit <= 0 {
		return defaultTipsetsPageLimit
	}
	return limit
}

func GetTipSetsList(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTipSetsList")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 分页参数归一化：0/负数不再等同于「取全量」。
	origLimit := req.Limit
	req.Limit = normalizeTipsetsLimit(req.Limit)
	if req.Index < 0 {
		req.Index = 0
	}
	if origLimit != req.Limit {
		alog.Infof("paging limit normalized: %d -> %d (index=%d)", origLimit, req.Limit, req.Index)
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetTotalCountForTipSets(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.TipSetStates
	}

	var tipSetRes []model.TipSetRes
	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.TipSetStates, countUtils, common2.TipsetsListAggregator, req, "Tipset")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &tipSetRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.TipSetListRes{TotalCount: totalCount, TipSets: tipSetRes}
	c.JSON(http.StatusOK, res)
}
