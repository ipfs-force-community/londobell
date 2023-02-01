package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	log = logging.Logger("server")
)

func Run(cctx *cli.Context, useAPI bool) error {
	router := gin.New()
	router.Use(CrosHandler())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.GET("/ping", Pong)

	var (
		err error
		ctx = context.Background()
	)

	if useAPI { // todo
		adapter.API = adapter.NewAppropriateAPI(cctx.StringSlice("apis"))
		err = adapter.API.Choose(ctx)
		if err != nil {
			return err
		}

		_, err = adapter.API.InjectNewFullNode(cctx)
		if err != nil {
			return err
		}

		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		go func() {
			for {
				select {
				case <-tick.C:
					err = adapter.API.Choose(ctx)
					if err != nil {
						log.Warn(err)
						continue
					}

					injectNew, err := adapter.API.InjectNewFullNode(cctx)
					if injectNew {
						if err != nil {
							log.Errorf("inject new fullnode failed: %v", err)
						} else {
							log.Info("inject new fullnode successfully")
						}
					} else {
						log.Info("no new fullnode injected")
					}
				}
			}
		}()

		RegisterAdapterApi(router)
	} else {
		aggregators.InitAggregators()
		mongoutil.InitDB()
		mongoutil.Client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoutil.DbConfig.URL).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
		if err != nil {
			return err
		}
		defer mongoutil.Client.Disconnect(ctx) //nolint:errcheck
		db := mongoutil.Client.Database(mongoutil.DbConfig.Name)
		mongoutil.TraceCol = db.Collection("ExecTrace")
		mongoutil.ActorBalanceCol = db.Collection("ActorBalance")
		mongoutil.FinalHeightCol = db.Collection("FinalHeight")
		mongoutil.MinerSectorHealthCol = db.Collection("MinerSectorHealth")
		mongoutil.TipSetCol = db.Collection("Tipset")
		mongoutil.ActorStateCol = db.Collection("ActorState")
		mongoutil.MinerFundsCol = db.Collection("MinerFunds")
		mongoutil.BlockHeaderCol = db.Collection("BlockHeader")
		mongoutil.ClaimedPowerCol = db.Collection("ClaimedPower")
		mongoutil.DealProposalCol = db.Collection("DealProposal")
		mongoutil.MessageCol = db.Collection("Message")
		mongoutil.MessageBlockCol = db.Collection("MessageBlock")

		// tmp
		mongoutil.TmpClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoutil.DbConfig.TmpURL).SetRegistry(bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build()))
		if err != nil {
			return err
		}
		defer mongoutil.TmpClient.Disconnect(ctx) //nolint:errcheck
		tmpDB := mongoutil.TmpClient.Database(mongoutil.DbConfig.TmpName)
		mongoutil.TmpTraceCol = tmpDB.Collection("ExecTrace")
		mongoutil.TmpTipSetCol = tmpDB.Collection("Tipset")
		mongoutil.TmpBlockHeaderCol = tmpDB.Collection("BlockHeader")
		mongoutil.TmpMessageCol = tmpDB.Collection("Message")
		mongoutil.TmpMessageBlockCol = tmpDB.Collection("MessageBlock")

		RegisterAggregatorsApi(router)
	}

	s := &http.Server{
		Addr:         fmt.Sprintf(":%s", cctx.String("port")),
		Handler:      router,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	//err = router.Run(":" + cctx.String("port"))
	//if err != nil {
	//	return err
	//}

	return err
}

func RegisterAdapterApi(router *gin.Engine) {
	group := router.Group("/adapter").Use()
	{
		group.POST("/actor", adapter.GetActorInfo)
		group.POST("/actors", adapter.GetActorsInfo)
		group.POST("/actor_ids", adapter.GetActorIDs)
		group.POST("/epoch", adapter.GetEpochInfo)
		group.POST("/miner", adapter.GetMinerInfo)
		group.POST("/sector", adapter.GetSectorInfo)
		group.POST("/batchminers", adapter.GetBatchMinersInfo)
		group.POST("/sectorpower", adapter.GetSectorPowerInfo)
		group.POST("/precommit_deposit_toburn", adapter.GetPreCommitDepositToBurnInfo)
		group.POST("/sector_for_miner", adapter.GetSectorForMinerInfo)
		group.POST("/mpool", adapter.GetPendingMessages)
		group.POST("/list_miners", adapter.GetListMiners)
		group.POST("/current_sector_initial_pledge", adapter.CurrentSectorInitialPledge)
		group.POST("/sectornumber_by_dealID", adapter.GetSectorNumberByDealID)
	}
}

func RegisterAggregatorsApi(router *gin.Engine) {
	group := router.Group("/aggregators").Use()
	{
		group.POST("/address", aggregators.GetAddress)
		group.POST("/agg_pre_netfee", aggregators.GetAggPreNetfee)
		group.POST("/agg_pro_netfee", aggregators.GetAggProNetfee)
		group.POST("/block", aggregators.GetBlock)
		group.POST("/miner_blockreward", aggregators.GetMinerBlockReward)
		group.POST("/miners_mined", aggregators.GetMinersMined)
		group.POST("/final_height", aggregators.GetFinalHeight)
		group.POST("/miners_info", aggregators.GetMinersInfo)
		group.POST("/multisig_message", aggregators.GetMultisigMessage)
		group.POST("/punishment", aggregators.GetPunishment)
		group.POST("/wincount", aggregators.GetWinCount)
		group.POST("/traces", aggregators.GetTraces)
		group.POST("/child_epoch", aggregators.GetChildEpoch)
		group.POST("/miners_blockreward", aggregators.GetMinersBlockReward)
		group.POST("/burn_monitor", aggregators.GetBurnMonitor)
		group.POST("/latest_tipset", aggregators.GetLatestTipSet)
		group.POST("/total_block_count", aggregators.GetTotalBlockCount)
		group.POST("/actor_state_epoch", aggregators.GetActorStateForEpoch)
		group.POST("/tipset", aggregators.GetTipSet)
		group.POST("/miner_info", aggregators.GetMinerInfo)
		group.POST("/balance", aggregators.GetBalance)
		group.POST("/miners_for_owner", aggregators.GetMinersForOwner)
		group.POST("/messages_for_actor", aggregators.GetMessagesForActor)
		group.POST("/transfer_messages", aggregators.GetTransferMessages)
		group.POST("/time_of_trace", aggregators.GetTimeOfTrace)
		group.POST("/create_time", aggregators.GetCreateTime)
		group.POST("/gascost_for_sector", aggregators.GetGasCostForSector)
		group.POST("/transfer_message_for_largeAmount", aggregators.GetTransferMessageForLargeAmount)
		group.POST("/deals", aggregators.GetDeals)
		group.POST("/detail_for_deal", aggregators.GetDetailForDeal)
		group.POST("/blockheader", aggregators.GetBlockHeader)
		group.POST("/trace_for_message", aggregators.GetTraceForMessage)
		group.POST("/batch_trace_for_message", aggregators.GetBatchTraceForMessage)
		group.POST("/child_transfers_for_message", aggregators.GetChildTransfersForMessage)
		group.POST("/all_owners", aggregators.GetAllOwners)
		group.POST("/parent_tipset", aggregators.GetParentTipSet)
		group.POST("/blockheader_by_cid", aggregators.GetBlockHeaderByCid)
		group.POST("/blockmessages_by_methodname", aggregators.GetBlockMessagesByMethodName)
		group.POST("/actormessages_by_methodname", aggregators.GetActorMessagesByMethodName)
		group.POST("/blockheaders_by_miner", aggregators.GetBlockHeadersByMiner)
		group.POST("/deals_by_addr", aggregators.GetDealsByAddr)
		group.POST("/all_methods", aggregators.GetAllMethods)
		group.POST("/all_methods_for_actor", aggregators.GetAllMethodsForActor)
		group.POST("/blocks_for_message", aggregators.GetBlocksForMessage)
		group.POST("/messages_for_block", aggregators.GetMessagesForBlock)
		group.POST("/count_and_methods_of_messages_for_blockheader", aggregators.GetCountAndMethodsOfMessagesForBlockHeader)
		group.POST("/blockheader_messages_by_methodname", aggregators.GetBlockHeaderMessagesByMethodName)
		group.POST("/richlist", aggregators.GetRichList)
	}
}

func CrosHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
		context.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma,token,openid,opentoken")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
		context.Header("Access-Control-Max-Age", "172800")
		context.Header("Access-Control-Allow-Credentials", "false")
		context.Set("content-type", "application/json") //// 设置返回格式是json

		if method == "OPTIONS" {
			context.JSON(http.StatusOK, nil)
		}

		//处理请求
		context.Next()
	}
}

func Pong(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
