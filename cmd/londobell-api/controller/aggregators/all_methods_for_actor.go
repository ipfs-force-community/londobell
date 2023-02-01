package aggregators

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetAllActorMethods(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAllActorMethods")
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

	ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	actorID := ID.String()[1:]
	curEpoch := common.GetCurEpoch()

	actorMsgsByMethodNameMap, err := multiquery.GetAllActorMsgsByMethodNameMap(ctx, &multiquery.DBStateManager, actorID, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var allMethods []model.AllMethodsRes
	for methodName, count := range actorMsgsByMethodNameMap {
		allMethods = append(allMethods, model.AllMethodsRes{MethodName: methodName, Count: count})
	}

	// create message
	actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
	if err != nil {
		log.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	switch {
	case addr == sbuiltin.SystemActorAddr, addr == sbuiltin.InitActorAddr, addr == sbuiltin.RewardActorAddr, addr == sbuiltin.CronActorAddr, addr == sbuiltin.StoragePowerActorAddr,
		addr == sbuiltin.VerifiedRegistryActorAddr, addr == sbuiltin.BurntFundsActorAddr, addr == sbuiltin.DatacapActorAddr, addr == sbuiltin.EthereumAddressManagerActorAddr,
		builtin.IsAccountActor(actor.Code), builtin.IsEthAccountActor(actor.Code), builtin.IsPlaceholderActor(actor.Code):
		res.Data = allMethods
		c.JSON(http.StatusOK, res)
		return
	case builtin.IsStorageMinerActor(actor.Code):
		// CreateMiner
		allMethods = append(allMethods, model.AllMethodsRes{MethodName: "CreateMiner", Count: 1})
	case builtin.IsEvmActor(actor.Code):
		// CreateExternal
		allMethods = append(allMethods, model.AllMethodsRes{MethodName: "CreateExternal", Count: 1})
	default:
		// Exec
		allMethods = append(allMethods, model.AllMethodsRes{MethodName: "Exec", Count: 1})
	}

	res.Data = allMethods
	c.JSON(http.StatusOK, res)
}
