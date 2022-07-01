package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/controller"
	"github.com/urfave/cli/v2"
)

func Run(cctx *cli.Context) error {
	router := gin.New()
	router.Use(CrosHandler())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.GET("/ping", Pong)

	err := controller.GetFullNodeAPI(cctx.Context)
	if err != nil {
		return err
	}

	RegisterApi(router)
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

func RegisterApi(router *gin.Engine) {
	group := router.Group("").Use()
	{
		group.POST("/actor", controller.GetActorInfo)
		group.POST("/epoch", controller.GetEpochInfo)
		group.POST("/miner", controller.GetMinerInfo)
		group.POST("/sector", controller.GetSectorInfo)
		group.POST("/batchminers", controller.GetBatchMinersInfo)
		group.POST("/sectorpower", controller.GetSectorPowerInfo)
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
