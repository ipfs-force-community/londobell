package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
)

var log = logging.Logger("server")

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

	if useAPI {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		go func() {
			for {
				select {
				case <-tick.C:
					err = adapter.ChooseAPI(cctx)
					if err != nil {
						log.Fatal(err)
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
	// todo: 更新文档
	group := router.Group("/adapter").Use()
	{
		group.POST("/actor", adapter.GetActorInfo)
		group.POST("/epoch", adapter.GetEpochInfo)
		group.POST("/miner", adapter.GetMinerInfo)
		group.POST("/sector", adapter.GetSectorInfo)
		group.POST("/batchminers", adapter.GetBatchMinersInfo)
		group.POST("/sectorpower", adapter.GetSectorPowerInfo)
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
