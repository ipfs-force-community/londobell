package aggregators

import (
	"encoding/json"
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
	"golang.org/x/net/context"
)

func GetMessagesForActor(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMessagesForActor")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetTotalCountForActorMsgs(ctx, req.Addr, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.Count
	}

	api := fullnode.API.GetAppropriateAPI()
	addrs, err := GetAllAddrs(ctx, req.Addr, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addrs = addrs

	var messagesForActor []model.MessageForActor

	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, countUtils, messagesForActorAggregator, req, "ExecTrace")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) != 0 {
			raw := multiResult
			rawByte, err := json.Marshal(raw)
			if err != nil {
				log.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = json.Unmarshal(rawByte, &messagesForActor)
			if err != nil {
				log.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}
	}

	// create message
	addr, err := address.NewFromString(buildnet.NetPrefix + req.Addr)
	if err != nil {
		log.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

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
		res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: messagesForActor}
		c.JSON(http.StatusOK, res)
		return
	case builtin.IsStorageMinerActor(actor.Code):
		// CreateMiner
		req.To = sbuiltin.StoragePowerActorAddr.String()[1:]
		req.Method = 2
	case builtin.IsEvmActor(actor.Code):
		// CreateExternal
		req.To = sbuiltin.EthereumAddressManagerActorAddr.String()[1:]
		req.Method = 4
	default:
		// Exec
		req.To = sbuiltin.InitActorAddr.String()[1:]
		req.Method = 2
	}

	ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	robust, err := GetRobustByID(ctx, api, ID, actor)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr = robust

	pipe, err := util.Parse(model.Ctx{Addr: req.Addr, To: req.To, Method: req.Method}, string(createMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var createMessage model.MessageForActor

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
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

		err = json.Unmarshal(rawByte, &createMessage)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	messagesForActor = append(messagesForActor, createMessage)
	totalCount++

	res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: messagesForActor}
	c.JSON(http.StatusOK, res)
}
