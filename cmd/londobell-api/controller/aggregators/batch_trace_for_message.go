package aggregators

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetBatchTraceForMessage(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetBatchTraceForMessage")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var traceForMessageRes []model.TraceForMessageRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch, Cids: req.Cids}, string(batchTraceForMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.TraceCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &traceForMessageRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// search from the temporary repository if not found
	toSearchCids := make([]string, 0)

	for _, cid := range req.Cids {
		found := false
		for _, result := range traceForMessageRes {
			if result.Cid == cid {
				found = true
				break
			}
		}
		if !found {
			toSearchCids = append(toSearchCids, cid)
		}
	}

	var tmpTraceForMessageRes []model.TraceForMessageRes
	tmpPipe, err := Parse(model.Ctx{Cids: toSearchCids}, string(traceForMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	tmpCur, err := mongoutil.TmpTraceCol.Aggregate(ctx, tmpPipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = tmpCur.All(ctx, &tmpTraceForMessageRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	traceForMessageRes = append(traceForMessageRes, tmpTraceForMessageRes...)
	// 去重
	resultMap := make(map[string]struct{})
	resultList := make([]model.TraceForMessageRes, 0)
	for _, result := range traceForMessageRes {
		if _, ok := resultMap[result.Cid]; !ok {
			resultList = append(resultList, result)
			resultMap[result.Cid] = struct{}{}
		}
	}

	res.Data = resultList
	c.JSON(http.StatusOK, res)
}
