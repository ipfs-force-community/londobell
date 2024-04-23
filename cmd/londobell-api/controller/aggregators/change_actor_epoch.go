package aggregators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
    },
	{
        $project: {
			"ActorID": "$Addr",
			"Code":    1,
			"Balance": 1,
			"Epoch":   1,
			"_id":     0,
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
	startEpoch := req.Epoch
	if startEpoch%120 != 0 {
		err = fmt.Errorf("epoch must be multiple of 120")
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var changedActorsRes []model.ActorStateRes

	var baseRes []model.ActorStateRes
	reqRes, err := getActorState(startEpoch, ctx, countUtils)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if startEpoch >= 120 {
		baseEpoch := req.Epoch - 120
		baseRes, err = getActorState(baseEpoch, ctx, countUtils)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	changedActorsRes = diffActorState(reqRes, baseRes)

	// var uniqueCheck = make(map[string]struct{})
	// var data []model.ActorStateRes
	// for _, v := range changedActorsRes {
	// 	if _, ok := uniqueCheck[v.ActorID]; !ok {
	// 		uniqueCheck[v.ActorID] = struct{}{}
	// 		data = append(data, v)
	// 	}
	// }
	res.Data = changedActorsRes
	c.JSON(http.StatusOK, res)
}

func getActorState(epoch int64, ctx context.Context, countUtils []multiquery.CountUtil) ([]model.ActorStateRes, error) {
	var actorStateRes []model.ActorStateRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: epoch}, epochAggregator)
	if err != nil {
		return nil, err
	}
	multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ActorState")

	if err != nil {
		return nil, err
	}

	if len(multiResult) == 0 {
		return nil, nil
	}

	raw := multiResult
	rawByte, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawByte, &actorStateRes)
	if err != nil {
		return nil, err
	}
	return actorStateRes, nil
}

func diffActorState(reqRes, baseRes []model.ActorStateRes) []model.ActorStateRes {
	if baseRes == nil {
		return tidyActorCode(reqRes)
	}
	var changedActorsRes []model.ActorStateRes
	base := make(map[string]model.ActorStateRes)
	current := make(map[string]model.ActorStateRes)
	for _, v := range baseRes {
		base[v.ActorID] = v
	}
	for _, v := range reqRes {
		current[v.ActorID] = v
	}

	for k, v := range current {
		if _, ok := base[k]; !ok {
			changedActorsRes = append(changedActorsRes, v)
		} else {
			if v.Balance != base[k].Balance {
				changedActorsRes = append(changedActorsRes, v)
			}
		}
	}
	return tidyActorCode(changedActorsRes)
}

func tidyActorCode(actors []model.ActorStateRes) []model.ActorStateRes {
	for index, actor := range actors {
		parts := strings.Split(actor.Code, "/")
		code := parts[len(parts)-1]
		actors[index].Code = code
	}
	return actors
}
