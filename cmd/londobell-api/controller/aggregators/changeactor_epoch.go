package aggregators

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

var epochAggregator string = `
[
    {
        $match: {
            Epoch: ctx.StartEpoch
        }
    }
]
`

func GetChangeActorForEpoch(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetChangeActorForEpoch")
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var changedActorsRes []model.ChangedActorRes

	// multi dbs query
	{
		pipe, err := util.Parse(model.Ctx{StartEpoch: req.Epoch}, epochAggregator)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ChangedActor")

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

		err = json.Unmarshal(rawByte, &changedActorsRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}
	var uniqueCheck = make(map[string]struct{})
	var data []model.ChangedActorRes
	for _, v := range changedActorsRes {
		if _, ok := uniqueCheck[v.ActorID]; !ok {
			uniqueCheck[v.ActorID] = struct{}{}
			data = append(data, v)
		}
	}
	res.Data = data
	c.JSON(http.StatusOK, res)
}
