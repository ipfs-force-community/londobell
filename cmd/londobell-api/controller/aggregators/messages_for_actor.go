package aggregators

import (
	"context"
	"encoding/json"
	"math"
	"net/http"

	"github.com/filecoin-project/lotus/api/v0api"
	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

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
		totalCount += countUtil.ActorStates
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
		//multiResult, err := multiquery.MultiPagingQuery(ctx, req.Index, req.Limit, multiquery.ActorStates, countUtils, messagesForActorAggregator, req, "ActorMessage")

		multiResult, err := multiquery.MultiBiSearch(ctx, req.Index*req.Limit, req.Limit, countUtils, actorMessageNoSkip,
			monitor.GetCountOfMessageForActorAggregator(), req, "ActorMessage", multiquery.ActorStates)
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

	// todo: Skip the query creation message temporarily
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

	if builtin.IsStorageMinerActor(actor.Code) || builtin.IsEvmActor(actor.Code) {
		totalCount++
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
		}
	}

	//createMessage, err := getCreateMessage(ctx, req.Addr, api, countUtils)
	//if err != nil {
	//	log.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//if createMessage != nil {
	//	totalCount++
	//	if int64(len(messagesForActor)) < req.Limit {
	//		messagesForActor = append(messagesForActor, *createMessage)
	//	}
	//}

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
		//to         string
		//method     uint64
		robust     string
		methodName string
	)

	// notice: 升级时方法名变化需注意
	switch {
	case addr == sbuiltin.SystemActorAddr, addr == sbuiltin.InitActorAddr, addr == sbuiltin.RewardActorAddr, addr == sbuiltin.CronActorAddr, addr == sbuiltin.StoragePowerActorAddr,
		addr == sbuiltin.VerifiedRegistryActorAddr, addr == sbuiltin.BurntFundsActorAddr, addr == sbuiltin.DatacapActorAddr, addr == sbuiltin.EthereumAddressManagerActorAddr,
		builtin.IsAccountActor(actor.Code), builtin.IsEthAccountActor(actor.Code), builtin.IsPlaceholderActor(actor.Code):
		return nil, nil
	case builtin.IsStorageMinerActor(actor.Code):
		// CreateMiner
		methodName = "CreateMiner"
		//to = sbuiltin.StoragePowerActorAddr.String()[1:]
		//method = 2
	case builtin.IsEvmActor(actor.Code):
		// CreateExternal
		methodName = "CreateExternal"
		//to = sbuiltin.EthereumAddressManagerActorAddr.String()[1:]
		//method = 4
	default:
		// Exec
		methodName = "Exec"
		//to = sbuiltin.InitActorAddr.String()[1:]
		//method = 2
	}

	ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	// 支持实时性，避免robust还未入库的情况
	idStr := ID.String()[1:]

	var pipe interface{}
	// evm actor可能没存f2，只存了f4
	if builtin.IsEvmActor(actor.Code) {
		id, err := address.IDFromAddress(ID)
		if err != nil {
			return nil, err
		}

		pipe, err = util.Parse(model.Ctx{ID: id, IDStr: idStr, MethodName: methodName /*To: to, Method: method*/}, string(createMessageAggregator))
		if err != nil {
			return nil, err
		}
	} else {
		robust, err = GetRobustByID(ctx, api, ID, actor)
		if err != nil {
			return nil, err
		}

		pipe, err = util.Parse(model.Ctx{Addr: robust, IDStr: idStr, MethodName: methodName /*To: to, Method: method*/}, string(createMessageAggregator))
		if err != nil {
			return nil, err
		}
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
