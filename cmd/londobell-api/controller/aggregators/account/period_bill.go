package account

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/account"

	"github.com/gin-gonic/gin"

	common2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetPeriodBill(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetPeriodBill")

	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr, err = common2.GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var (
		periodIncome  account.PeriodIncome
		periodPay     account.PeriodPay
		periodGasCost account.PeriodGasCost
	)
	// multi dbs query
	{
		req.TransferType = "to"
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.AccountPeriodTransferAggregator, req, "ActorMessage")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) > 0 {
			raw := multiResult[0]
			rawByte, err := json.Marshal(raw)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = json.Unmarshal(rawByte, &periodIncome)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}
	}

	// multi dbs query
	{
		req.TransferType = "from"
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.AccountPeriodTransferAggregator, req, "ActorMessage")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) > 0 {
			raw := multiResult[0]
			rawByte, err := json.Marshal(raw)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = json.Unmarshal(rawByte, &periodPay)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}
	}

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, common2.AccountPeriodGasCostAggregator, req, "ActorMessage")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) > 0 {
			raw := multiResult[0]
			rawByte, err := json.Marshal(raw)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = json.Unmarshal(rawByte, &periodGasCost)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}
	}

	res.Data = account.PeriodBillRes{Income: periodIncome.TotalValue, Pay: periodPay.TotalValue, GasCost: periodGasCost.TotalGasCost}
	c.JSON(http.StatusOK, res)
}
