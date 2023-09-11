package miner

//func GetPeriodSectorsDiff(c *gin.Context) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	alog := log.With("miner", "GetPeriodSectorsDiff")
//
//	req := model.CommonReq{}
//	res := model.CommonRes{Code: model.Success}
//	err := c.BindJSON(&req)
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	curEpoch := common.GetCurEpoch()
//
//	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
//	if err != nil {
//		alog.Error(err)
//		util.ReturnOnErr(c, err)
//		return
//	}
//
//	var periodSectorsDiffRes miner.PeriodSectorsDiffRes
//	// multi dbs query
//	{
//		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.MinerPeriodSectorsDiffAggregator, req, "MinerSectorHealth")
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
//		raw := multiResult[0]
//		rawByte, err := json.Marshal(raw)
//		if err != nil {
//			alog.Error(err)
//			util.ReturnOnErr(c, err)
//			return
//		}
//
//		err = json.Unmarshal(rawByte, &periodSectorsDiffRes)
//		if err != nil {
//			alog.Error(err)
//			util.ReturnOnErr(c, err)
//			return
//		}
//	}
//
//	res.Data = periodSectorsDiffRes
//	c.JSON(http.StatusOK, res)
//}
