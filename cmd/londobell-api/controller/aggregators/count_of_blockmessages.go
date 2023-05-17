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

func GetCountOfBlockMessages(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetCountOfBlockMessages")

	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	//
	//blockFilter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gte", Value: req.StartEpoch}}}, {Key: "Epoch", Value: bson.D{{Key: "$lt", Value: req.EndEpoch}}}, {Key: "Depth", Value: 1}, {Key: "$or", Value: []bson.M{{"Msg.From": bson.D{{Key: "$regex", Value: "^1"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^3"}}}, {"Msg.From": bson.D{{Key: "$regex", Value: "^4"}}}}}}
	//count, err = col.CountDocuments(ctx, blockFilter)
	//if err != nil {
	//	return err
	//}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalCount := int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.Count
	}

	js := []byte(" [\n    {\n        $match: {\n            $and: [\n                {\"Depth\": 1},\n                {\"Epoch\": {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch}},\n                {$or: [{\"Msg.From\":{$regex: /^1/}}, {\"Msg.From\":{$regex: /^3/}}, {\"Msg.From\":{$regex: /^4/}}]},\n            ]\n        }\n    },\n        {\n            $group: {\n                _id: 0,\n                Count: {$sum:1}\n            }\n        }\n        ]")

	var blockCount []model.BlockCountRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, js, req, "ExecTrace")
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

		err = json.Unmarshal(rawByte, &blockCount)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	count := int64(0)
	for _, bc := range blockCount {
		count += bc.Count
	}
	res.Data = model.BlockCountRes{Count: count}
	c.JSON(http.StatusOK, res)
}
