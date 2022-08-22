package aggregators

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetAddress(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetAddress")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	//todo: 🍬？
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var addressRes []model.AddressRes

	pipe, err := Parse(model.Ctx{Addr: req.Addr}, string(addressAggregator))
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur, err := mongoutil.ActorBalanceCol.Aggregate(ctx, pipe)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	err = cur.All(ctx, &addressRes)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if len(addressRes) != 1 {
		util.ReturnOnErr(c, alog, fmt.Errorf("get wrong result, length of result shoule be one but is %v", len(addressRes)))
		return
	}

	res.Data = addressRes[0]
	c.JSON(http.StatusOK, res)
}
