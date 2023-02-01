package aggregators

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

func GetChildTransfersForMessage(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetChildTransfersForMessage")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
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

	pipe, err := util.Parse(model.Ctx{Cid: req.Cid}, string(childTransfersForMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var childTransfersForMessageRes []model.ChildTransfersForMessageRes
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

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &childTransfersForMessageRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = childTransfersForMessageRes
	c.JSON(http.StatusOK, res)
}
