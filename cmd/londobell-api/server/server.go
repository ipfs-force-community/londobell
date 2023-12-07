package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/gastracker"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/stats"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/contract"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/transaction"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/account"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/block"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/miner"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/dep"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/gin-gonic/gin"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/api"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	caccount "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/account"
	cminer "github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/miner"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

var (
	log = logging.Logger("server")
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body string
		url := c.Request.URL
		header := c.Request.Header
		ip := c.ClientIP()

		if c.Request.Method != http.MethodGet {
			var buf bytes.Buffer
			tee := io.TeeReader(c.Request.Body, &buf)
			bodybuf, _ := io.ReadAll(tee)
			c.Request.Body = io.NopCloser(&buf)
			body = string(bodybuf)
		}
		log.Infof("get request,url: %s,ip: %s,body: %s,header: %s", url, ip, body, header)
		c.Next()
	}
}

func Run(cctx *cli.Context, adapter bool) error {
	router := gin.New()
	router.Use(CrosHandler(), gin.Logger(), gin.Recovery(), RequestLogger())
	router.GET("/ping", Pong)

	var (
		err error
		ctx = context.Background()
	)

	if err := util.ParseNodes(cctx.String("nodeconfig")); err != nil {
		return err
	}

	fullnode.API = fullnode.NewAppropriateAPI(util.Nodes)
	err = fullnode.API.Choose(ctx)
	if err != nil {
		return err
	}

	if adapter {
		_, err = fullnode.API.InjectNewFullNode(cctx)
		if err != nil {
			return err
		}

		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		go func() {
			for {
				select {
				case <-tick.C:
					err = fullnode.API.Choose(ctx)
					if err != nil {
						log.Warn(err)
						continue
					}

					injectNew, err := fullnode.API.InjectNewFullNode(cctx)
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

		httpStoper, errCh := serveHTTP(fmt.Sprintf(":%s", cctx.String("port")), router)
		select {
		case err = <-errCh:

		case <-time.After(time.Duration(5)):

		}
		if err != nil {
			return fmt.Errorf("start http server: %w", err)
		}

		shutdownCh := make(chan struct{})
		doneCh := node.MonitorShutdown(
			shutdownCh,
			node.ShutdownHandler{Component: "http server", StopFunc: httpStoper},
		)

		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		select {
		case sig := <-sigCh:
			log.Warnw("received shutdown", "signal", sig)
		case <-doneCh:
			log.Warn("received shutdown")
		}

		os.Exit(1)
	} else {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		go func() {
			for range tick.C {
				err = fullnode.API.Choose(ctx)
				if err != nil {
					log.Warn(err)
					continue
				}

			}
		}()

		shutdownCh := make(chan struct{})

		var multiNode api.MultiNodeAPI
		stopper, err := dix.New(
			cctx.Context,
			dep.MultiQuery(context.TODO(), &multiquery.DBStateManager, &multiNode),
			dep.InjectRepoPath(cctx),
			dix.Override(new(dtypes.ShutdownChan), shutdownCh),
		)
		if err != nil {
			log.Error("stopper", err)
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		repoPath, err := dep.GetRepoPath(cctx)
		if err != nil {
			log.Error(err)
			return err
		}

		err = multiquery.FirstLoad(cctx.Context, &multiquery.DBStateManager)
		if err != nil {
			log.Error(err)
			return err
		}

		// 定时任务
		// 刷新agg config
		go multiquery.Reload(cctx.Context, &multiquery.DBStateManager, dep.ConfigFilePath(repoPath))

		// 刷新消息池
		go aggregators.RefreshMpool()

		// 刷新state索引
		mlog := log.With("server", "multi-query")
		go multiquery.PeriodicRefreshDataBaseState(cctx.Context, mlog, &multiquery.DBStateManager)

		// http server
		common.InitAggregators()
		RegisterAggregatorsApi(router)

		httpStoper, errCh := serveHTTP(fmt.Sprintf(":%s", cctx.String("port")), router)
		select {
		case err = <-errCh:

		case <-time.After(time.Duration(5)):

		}
		if err != nil {
			return fmt.Errorf("start http server: %w", err)
		}
		doneCh := node.MonitorShutdown(
			shutdownCh,
			node.ShutdownHandler{Component: "http server", StopFunc: httpStoper},
			node.ShutdownHandler{Component: "application", StopFunc: node.StopFunc(stopper)},
		)

		addr := cctx.String("RPCListen")
		if addr == "" {
			addr = multiquery.DefaultRPCListenAddr
		}
		endpoint, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("parse addr: %s, err: %v", addr, err)
		}
		return serveRPC(&multiNode, stopper, endpoint, doneCh, 0)
	}

	//s := &http.Server{
	//	Addr:         fmt.Sprintf(":%s", cctx.String("port")),
	//	Handler:      router,
	//	ReadTimeout:  time.Minute,
	//	WriteTimeout: time.Minute,
	//}
	//if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	//	log.Fatal(err)
	//}

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
		group.POST("/last_epoch", adapter.GetLastEpochInfo)
		group.POST("/miner", adapter.GetMinerInfo)
		group.POST("/sector", adapter.GetSectorInfo)
		group.POST("/batchminers", adapter.GetBatchMinersInfo)
		group.POST("/sectorpower", adapter.GetSectorPowerInfo)
		group.POST("/precommit_deposit_toburn", adapter.GetPreCommitDepositToBurnInfo)
		group.POST("/sector_for_miner", adapter.GetSectorForMinerInfo)
		group.POST("/mpool", adapter.GetPendingMessages)
		group.POST("/allmethods_for_mpool", adapter.GetAllMethodsForPendingMessages)
		group.POST("/list_miners", adapter.GetListMiners)
		group.POST("/current_sector_initial_pledge", adapter.CurrentSectorInitialPledge)
		group.POST("/sectornumber_by_dealID", adapter.GetSectorNumberByDealID)
		group.POST("/changed_actors", adapter.GetStateChaingedActors)
		group.POST("/version", adapter.GetVersion)
		group.POST("/initcode_for_evm", adapter.GetInitCodeForEvm)
		group.POST("/balance_at_market", adapter.GetBalanceAtMarket)
		group.POST("/active_sectors", adapter.GetActiveSectors)
		//group.POST("/sector_expiration", adapter.GetSectorExpiration)
	}
}

// todo: 范围查询 和 分页查询MultiPagingQuery
func RegisterAggregatorsApi(router *gin.Engine) {
	group := router.Group("/aggregators").Use()
	{
		// todo: 1. 原来范围请求变成分页请求的接口 2. 原来只请求formal，现在为了实时性也请求tmp 和天佑沟通
		group.POST("/address", aggregators.GetAddress)
		group.POST("/changed_actors", aggregators.GetChangeActorForEpoch)
		group.POST("/actor_state_epoch", aggregators.GetActorStateForEpoch) // todo: account只存一次，head主键不变
		group.POST("/balance", aggregators.GetBalance)
		group.POST("/richlist", aggregators.GetRichList)
		group.POST("/agg_pre_netfee", aggregators.GetAggPreNetFee)
		group.POST("/agg_pro_netfee", aggregators.GetAggProNetFee)
		group.POST("/block", aggregators.GetBlock)
		group.POST("/count_of_blockmessages", aggregators.GetCountOfBlockMessages)
		group.POST("/traces", aggregators.GetTraces) // only tianyou
		group.POST("/trace_for_message", aggregators.GetTraceForMessage)
		group.POST("/batch_trace_for_message", aggregators.GetBatchTraceForMessage)
		group.POST("/child_transfers_for_message", aggregators.GetChildTransfersForMessage)
		group.POST("/multisig_message", aggregators.GetMultisigMessage) // only tianyou
		group.POST("/miner_blockreward", aggregators.GetMinerBlockReward)
		group.POST("/miners_blockreward", aggregators.GetMinersBlockReward)
		group.POST("/miners_mined", aggregators.GetMinersMined)
		group.POST("/wincount", aggregators.GetWinCount) // todo: 全网至今总wincount
		group.POST("/wincount_for_miner", aggregators.GetWinCountForMiner)
		group.POST("/total_block_count", aggregators.GetTotalBlockCount) // todo: 全网至今总爆块数
		group.POST("/miners_for_owner", aggregators.GetMinersForOwner)   // only query from formal
		group.POST("/all_owners", aggregators.GetAllOwners)              // only query from formal
		group.POST("/miner_info", aggregators.GetMinerInfo)
		group.POST("/miners_info", aggregators.GetMinersInfo)
		group.POST("/gascost_for_sector", aggregators.GetGasCostForSector)
		group.POST("/burn_monitor", aggregators.GetBurnMonitor)
		group.POST("/punishment", aggregators.GetPunishment)
		group.POST("/final_height", aggregators.GetFinalHeight)
		group.POST("/latest_tipset", aggregators.GetLatestTipSet)
		group.POST("/child_epoch", aggregators.GetChildEpoch)
		group.POST("/tipset", aggregators.GetTipSet)
		group.POST("/tipsets", aggregators.GetTipSets)
		group.POST("/parent_tipset", aggregators.GetParentTipSet)
		group.POST("/latest_time_of_trace", aggregators.GetLatestTimeOfTrace)
		group.POST("/create_time", aggregators.GetCreateTime)
		group.POST("/deals", aggregators.GetDeals)
		group.POST("/deal_by_id", aggregators.GetDealByID)
		group.POST("/detail_for_deal", aggregators.GetDetailForDeal)
		group.POST("/deals_by_addr", aggregators.GetDealsByAddr)
		group.POST("/blockmessages_by_methodname", aggregators.GetBlockMessagesByMethodName)
		group.POST("/actormessages_by_methodname", aggregators.GetActorMessagesByMethodName)
		group.POST("/messages_for_actor", aggregators.GetMessagesForActor)
		group.POST("/transfer_messages", aggregators.GetTransferMessages)
		group.POST("/transfer_message_for_largeAmount", aggregators.GetTransferMessageForLargeAmount)
		group.POST("/blockheader", aggregators.GetBlockHeader)
		group.POST("/incoming_blockheader", aggregators.GetIncomingBlockHeader)
		group.POST("/blockheader_by_cid", aggregators.GetBlockHeaderByCid)
		group.POST("/incoming_blockheader_by_cid", aggregators.GetIncomingBlockHeaderByCid)
		group.POST("/blockheaders_by_miner", aggregators.GetBlockHeadersByMiner) // 出块列表，出块奖励额外获取
		//group.POST("/mined_by_miner_range", aggregators.GetMinedByMinerForRange)
		group.POST("/blocks_for_message", aggregators.GetBlocksForMessage) // todo: epoch可不要，遍历查询即可
		group.POST("/count_and_methods_of_messages_for_blockheader", aggregators.GetCountAndMethodsOfMessagesForBlockHeader)
		group.POST("/messages_for_block", aggregators.GetMessagesForBlock)
		group.POST("/blockheader_messages_by_methodname", aggregators.GetBlockHeaderMessagesByMethodName)
		group.POST("/all_methods", aggregators.GetAllMethods)
		group.POST("/all_methods_for_actor", aggregators.GetAllActorMethods)
		group.POST("/version", aggregators.GetVersion)
		group.POST("/get_transaction_by_cid", aggregators.GetTransactionByCid)
		group.POST("/get_transaction_receipt_by_cid", aggregators.GetTransactionReceiptByCid)
		group.POST("/initcode_for_evm", aggregators.GetInitCodeForEvm)
		group.POST("/messagecid_by_hash", aggregators.GetMessageCidByHash)
		group.POST("/hash_by_messagecid", aggregators.GetHashByMessageCid)
		group.POST("/state_final_height", aggregators.GetStateFinalHeight)
		group.POST("/child_calls_for_message", aggregators.GetChildCallsForMessage)
		group.POST("/events_for_actor", aggregators.GetEventsForActor)
		group.POST("/events_for_message", aggregators.GetEventsForMessage)
		group.POST("/events_for_epochrange", aggregators.GetEventsForEpochRange)
		group.POST("/tipsets_list", aggregators.GetTipSetsList)
		group.POST("/mpool", aggregators.GetMpool)
		group.POST("/allmethods_for_mpool", aggregators.GetAllMethodsForPendingMessages)
		group.POST("/blockmessages_for_epochrange", aggregators.GetBlockMessagesForEpochRange)

	}

	minergroup := router.Group("/aggregators/miner").Use()
	{
		minergroup.POST("/periodblockrewards", cminer.GetPeriodBlockRewards)
		minergroup.POST("/periodwincounts", cminer.GetPeriodWinCounts)
		minergroup.POST("/periodgascost", cminer.GetPeriodGasCosts)
		minergroup.POST("/periodgascostforpublishdeals", cminer.GetPeriodGasCostsForPublishDeals)
		minergroup.POST("/periodpunishments", cminer.GetPeriodPunishments)
		//minergroup.POST("/periodsectorsdiff", cminer.GetPeriodSectorsDiff) // 分库有问题
		//minergroup.POST("/periodpledgediff", cminer.GetPeriodPledgeDiff)   // 分库有问题
		minergroup.POST("/sectorhealthhistory", cminer.GetSectorHealthHistory)
		minergroup.POST("/pledgehistory", cminer.GetPledgeHistory)
		minergroup.POST("/periodexpirations", cminer.GetPeriodSectorExpirations)
		minergroup.POST("/qapowerhistory", cminer.GetQAPowerHistory)

		minergroup.POST("/hourlysectorhealthlist", miner.NewMockSectorHealthRes().GetMockResponse)
		minergroup.POST("/hourlypledgelist", miner.NewMockPledgeRes().GetMockResponse)
		minergroup.POST("/livesectorlist", nil)
		minergroup.POST("/sectorclaims", miner.NewMockClaimRes().GetMockResponse)
		minergroup.POST("/periodclaimexpiration", miner.NewMockClaimRes().GetMockResponse)
		minergroup.POST("/expiredclaims", miner.NewMockClaimRes().GetMockResponse)
		minergroup.POST("/sectordetail", nil)
		minergroup.POST("/claimdetail", miner.NewMockClaimRes().GetMockResponse)
	}

	accountgroup := router.Group("/aggregators/account").Use()
	{
		accountgroup.POST("/balancehistory", account.NewMockBalanceRes().GetMockResponse)
		accountgroup.POST("/balancelist", account.NewMockBalanceListRes().GetMockResponse)
		accountgroup.POST("/messagelist", account.NewMockMessageListRes().GetMockResponse)
		accountgroup.POST("/messagelistimplicit", account.NewMockMessageListRes().GetMockResponse)
		accountgroup.POST("/transferlist", account.NewMockBriefMessageListRes().GetMockResponse)
		//accountgroup.POST("/transferobjectlist", account.NewMockBriefMessageListRes().GetMockResponse)
		accountgroup.POST("/periodbill", caccount.GetPeriodBill)
	}

	blockgroup := router.Group("/aggregators/block").Use()
	{
		blockgroup.POST("/tipsethistory", block.NewMockTipSetRes().GetMockResponse)
		blockgroup.POST("/tipsetlist", block.NewMockTipSetListRes().GetMockResponse)
		blockgroup.POST("/blockdetail", block.NewMockBlockHeaderRes().GetMockResponse)
		blockgroup.POST("/blockreward", block.NewMockBlockRewardRes().GetMockResponse)
		blockgroup.POST("/packedmessages", account.NewMockMessageListRes().GetMockResponse)
		blockgroup.POST("/blkcountlist", block.NewMockBlockCountListRes().GetMockResponse)
	}

	contractgroup := router.Group("/aggregators/contract").Use()
	{
		contractgroup.POST("/initcode", contract.NewMockInitCodeRes().GetMockResponse)
		contractgroup.POST("/creation", contract.NewMockCreationRes().GetMockResponse)
		contractgroup.POST("/eventlist", contract.NewMockEventListRes().GetMockResponse)
	}

	transactiongroup := router.Group("/aggregators/transaction").Use()
	{
		transactiongroup.POST("/messagedetail", account.NewMockMessageRes().GetMockResponse)
		transactiongroup.POST("/messageinternals", account.NewMockMessageListRes().GetMockResponse)
		transactiongroup.POST("/messagelist", account.NewMockMessageListRes().GetMockResponse)
		transactiongroup.POST("/txbymethodlist", account.NewMockBriefMessageListRes().GetMockResponse)
		transactiongroup.POST("/getreceipt", account.NewMockMessageListRes().GetMockResponse)
		transactiongroup.POST("/txhash", transaction.NewMockTxHashRes().GetMockResponse)
	}

	statsgroup := router.Group("/aggregators/stats").Use()
	{
		statsgroup.POST("/activeminers", stats.NewMockActiveMinerRes().GetMockResponse)
		statsgroup.POST("/newactors", stats.NewMockNewActorRes().GetMockResponse)
	}

	gastrackergroup := router.Group("/aggregators/gastracker").Use()
	{
		gastrackergroup.POST("averagegascostlist", gastracker.NewMockAverageGasCostListRes().GetMockResponse)
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
