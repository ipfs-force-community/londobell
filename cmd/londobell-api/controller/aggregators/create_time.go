package aggregators

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"

	"github.com/filecoin-project/go-address"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/build/buildconstants"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetCreateTime(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetCreateTime")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	addr, err := address.NewFromString(buildnet.NetPrefix + req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	api := fullnode.API.GetAppropriateAPI()
	actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	switch {
	case addr == sbuiltin.SystemActorAddr, addr == sbuiltin.InitActorAddr, addr == sbuiltin.RewardActorAddr, addr == sbuiltin.CronActorAddr,
		addr == sbuiltin.StoragePowerActorAddr, addr == sbuiltin.VerifiedRegistryActorAddr, addr == sbuiltin.BurntFundsActorAddr:
		res.Data = model.TimeOfTraceRes{Epoch: 0}
		c.JSON(http.StatusOK, res)
		return
	case addr == sbuiltin.DatacapActorAddr:
		res.Data = model.TimeOfTraceRes{Epoch: buildconstants.UpgradeSharkHeight}
		c.JSON(http.StatusOK, res)
		return
	case addr == sbuiltin.EthereumAddressManagerActorAddr:
		res.Data = model.TimeOfTraceRes{Epoch: buildconstants.UpgradeHyggeHeight}
		c.JSON(http.StatusOK, res)
		return
	case builtin.IsAccountActor(actor.Code), builtin.IsEthAccountActor(actor.Code), builtin.IsPlaceholderActor(actor.Code):
		countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		req.Addr, err = common.GetIDByAddr(ctx, req.Addr)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		pipe, err := util.Parse(model.Ctx{StartEpoch: 0, EndEpoch: math.MaxInt64, Addr: req.Addr, Sort: 1}, string(common.TimeOfTraceAggregator))
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		var createTimeRes []model.TimeOfTraceRes

		// multi dbs query
		{
			multiResult, err := multiquery.MultiUnionQuery(ctx, pipe, countUtils, "ActorMessage")
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

			err = json.Unmarshal(rawByte, &createTimeRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			// 逆序排序
			sort.Slice(createTimeRes, func(i, j int) bool {
				return createTimeRes[i].Epoch < createTimeRes[j].Epoch
			})

			if len(createTimeRes) == 0 {
				c.JSON(http.StatusOK, res)
				return
			}

			res.Data = createTimeRes[0]
			c.JSON(http.StatusOK, res)
			return
		}
	// case builtin.IsStorageMinerActor(actor.Code):
	// CreateMiner
	// req.MethodName = "CreateMiner"
	//req.To = sbuiltin.StoragePowerActorAddr.String()[1:]
	//req.Method = 2
	// case builtin.IsEvmActor(actor.Code):
	// CreateExternal
	// req.MethodName = "CreateExternal"
	//req.To = sbuiltin.EthereumAddressManagerActorAddr.String()[1:]
	//req.Method = 4
	default:
		// Exec
		// req.MethodName = "Exec"
		//req.To = sbuiltin.InitActorAddr.String()[1:]
		//req.Method = 2
		// CreateMsgTimeAggregator := `
		// [
		// 	{
		// 		$match: {
		// 			"ActorID": ctx.Addr,
		// 			"IsBlock": true,
		// 		}
		// 	},
		// 	{
		// 		$sort: {
		// 			"Epoch": 1  // 创建表理论上含有ActorID的doc只有一个
		// 		}
		// 	},
		// 	{
		// 		$limit: 1
		// 	},
		// 	{
		// 		$project: {
		// 			_id: 0,
		// 			Epoch: "$Epoch"
		// 		}
		// 	}
		// ]
		// `

		pipe, err := util.Parse(model.Ctx{Addr: req.Addr}, common.TimeOfCreateAggregator)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		var createTimeRes model.TimeOfTraceRes

		// multi dbs query
		{
			multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "CreateMessage")
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			if len(multiResult) == 0 {
				c.JSON(http.StatusOK, res)
				return
			}

			raw := multiResult[0]
			rawByte, err := json.Marshal(raw)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = json.Unmarshal(rawByte, &createTimeRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}

		res.Data = createTimeRes
		c.JSON(http.StatusOK, res)
	}

	// ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	// if err != nil {
	// 	alog.Error(err)
	// 	util.ReturnOnErr(c, err)
	// 	return
	// }

	// idStr := ID.String()[1:]
	// if builtin.IsEvmActor(actor.Code) {
	// 	id, err := address.IDFromAddress(ID)
	// 	if err != nil {
	// 		alog.Error(err)
	// 		util.ReturnOnErr(c, err)
	// 		return
	// 	}

	// 	pipe, err = util.Parse(model.Ctx{ID: id, IDStr: idStr, MethodName: req.MethodName}, string(createMessageAggregator))
	// 	if err != nil {
	// 		alog.Error(err)
	// 		util.ReturnOnErr(c, err)
	// 		return
	// 	}
	// } else {
	// 	robust, err := GetRobustByID(ctx, api, ID, actor)
	// 	if err != nil {
	// 		alog.Error(err)
	// 		util.ReturnOnErr(c, err)
	// 		return
	// 	}

	// 	req.Addr = robust
	// 	pipe, err = util.Parse(model.Ctx{Addr: req.Addr, IDStr: idStr, MethodName: req.MethodName}, string(createTimeAggregator))
	// 	if err != nil {
	// 		alog.Error(err)
	// 		util.ReturnOnErr(c, err)
	// 		return
	// 	}
	// }

}
