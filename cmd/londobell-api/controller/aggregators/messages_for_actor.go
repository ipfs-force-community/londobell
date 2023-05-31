package aggregators

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/filecoin-project/lotus/api/v0api"

	"github.com/filecoin-project/go-address"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
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
	actorID, err := GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr = actorID

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	var messagesForActor []model.MessageForActor

	// multi dbs query
	{
		multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, countUtils, messagesForActorAggregator, req, "ActorMessage")
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

	if int64(len(messagesForActor)) < req.Limit {
		createMessage, err := getCreateMessage(ctx, req.Addr, api, countUtils)
		if err != nil {
			log.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if createMessage != nil {
			messagesForActor = append(messagesForActor, *createMessage)
			totalCount++
		}
	}

	res.Data = model.MessagesForActorRes{TotalCount: totalCount, MessagesForActor: messagesForActor}
	c.JSON(http.StatusOK, res)
}

func getCreateMessage(ctx context.Context, addrReq string, api v0api.FullNode, countUtils []multiquery.CountUtil) (*model.MessageForActor, error) {
	// create message
	addr, err := address.NewFromString(buildnet.NetPrefix + addrReq)
	if err != nil {
		return nil, err
	}

	actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	var (
		to     string
		method uint64
		robust string
	)

	switch {
	case addr == sbuiltin.SystemActorAddr, addr == sbuiltin.InitActorAddr, addr == sbuiltin.RewardActorAddr, addr == sbuiltin.CronActorAddr, addr == sbuiltin.StoragePowerActorAddr,
		addr == sbuiltin.VerifiedRegistryActorAddr, addr == sbuiltin.BurntFundsActorAddr, addr == sbuiltin.DatacapActorAddr, addr == sbuiltin.EthereumAddressManagerActorAddr,
		builtin.IsAccountActor(actor.Code), builtin.IsEthAccountActor(actor.Code), builtin.IsPlaceholderActor(actor.Code):
		return nil, nil
	case builtin.IsStorageMinerActor(actor.Code):
		// CreateMiner
		to = sbuiltin.StoragePowerActorAddr.String()[1:]
		method = 2
	case builtin.IsEvmActor(actor.Code):
		// CreateExternal
		to = sbuiltin.EthereumAddressManagerActorAddr.String()[1:]
		method = 4
	default:
		// Exec
		to = sbuiltin.InitActorAddr.String()[1:]
		method = 2
	}

	ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	robust, err = GetRobustByID(ctx, api, ID, actor)
	if err != nil {
		return nil, err
	}

	pipe, err := util.Parse(model.Ctx{Addr: robust, To: to, Method: method}, string(createMessageAggregator))
	if err != nil {
		return nil, err
	}

	var createMessage model.MessageForActor

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
		if err != nil {
			return nil, err
		}

		if len(multiResult) == 0 {
			// todo: 未找到就先不管了
			return nil, nil
		}

		raw := multiResult[0]
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawByte, &createMessage)
		if err != nil {
			return nil, err
		}
	}

	return &createMessage, nil
}
