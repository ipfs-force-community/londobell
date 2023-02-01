package aggregators

//
//import (
//	"encoding/json"
//	"net/http"
//
//	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"
//
//	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
//
//	"github.com/gin-gonic/gin"
//	"context"
//
//	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
//	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
//)
//
//func GetEvmInitCodeByActorID(c *gin.Context) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	alog := log.With("method", "GetEvmInitCodeByActorID")
//	req := model.CommonReq{}
//	res := model.CommonRes{Code: model.Success}
//	err := c.BindJSON(&req)
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	req.Addr, err = GetIDByAddr(ctx, req.Addr)
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	pipe, err := util.Parse(model.Ctx{Addr: req.Addr}, monitor.GetEvmInitCodeByActorIDAggregator())
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	var evmInitCodeByActorIDRes []model.GetEvmInitCodeByActorIDRes
//
//	// multi dbs query
//	{
//		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "EvmInitCode")
//		if err != nil {
//			alog.Error(err)
//			util.ReturnOnErr(c, err)
//			return
//		}
//
//		if len(multiResult) == 0 {
//			c.JSON(http.StatusOK, res)
//			return
//		}
//
//		raw := multiResult
//		rawByte, err := json.Marshal(raw)
//		if err != nil {
//			alog.Error(err)
//			util.ReturnOnErr(c, err)
//			return
//		}
//
//		err = json.Unmarshal(rawByte, &evmInitCodeByActorIDRes)
//		if err != nil {
//			alog.Error(err)
//			util.ReturnOnErr(c, err)
//			return
//		}
//	}
//
//	if len(evmInitCodeByActorIDRes) == 0 {
//		c.JSON(http.StatusOK, res)
//		return
//	}
//
//	res.Data = evmInitCodeByActorIDRes[0]
//
//	c.JSON(http.StatusOK, res)
//}
