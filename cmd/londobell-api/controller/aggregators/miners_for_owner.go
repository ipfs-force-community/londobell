package aggregators

import (
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
)

// todo: 请求高度区间, formal db每个高度都存，只从formal获取即可
func GetMinersForOwner(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMinersForOwner")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	formal := multiquery.DBStateManager.GetFormalCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(formal.Url())
	if !ok {
		alog.Error(fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url()))
		util.ReturnOnErr(c, err)
		return
	}

	latestEpoch, err := GetLatestEpoch(ctx, cols, "MinerFunds")
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var minersForOwnerRes []model.MinersForOwnerRes
	pipe, err := util.Parse(model.Ctx{StartEpoch: int64(latestEpoch), EndEpoch: int64(latestEpoch) + 1, Addr: req.Addr}, string(minersForOwnerAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "MinerFunds" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			err = cur.All(ctx, &minersForOwnerRes)
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}

			res.Data = minersForOwnerRes
			c.JSON(http.StatusOK, res)

			return
		}
	}

	c.JSON(http.StatusOK, res)
}

// todo: 目前只适用Epoch字段名的
func GetLatestEpoch(ctx context.Context, cols multiquery.Collections, tableName string) (abi.ChainEpoch, error) {
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{
					Key: "Epoch", Value: bson.D{{Key: "$gt", Value: 0}},
				},
			},
		},
	}
	sortStage := bson.D{
		{
			Key: "$sort", Value: bson.D{{Key: "Epoch", Value: -1}},
		},
	}
	limitStage := bson.D{
		{
			Key: "$limit", Value: 1,
		},
	}
	projectStage := bson.D{
		{
			Key: "$project", Value: bson.D{{Key: "_id", Value: 0}, {Key: "Epoch", Value: "$Epoch"}},
		},
	}

	var res []struct {
		Epoch abi.ChainEpoch
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			cursor, err := col.Aggregate(ctx, mongo.Pipeline{matchStage, sortStage, limitStage, projectStage})
			if err != nil {
				return 0, err
			}

			if err = cursor.All(ctx, &res); err != nil {
				return 0, err
			}

			if len(res) != 1 {
				return 0, fmt.Errorf("get wrong result, length of result shoule be one")
			}

			return res[0].Epoch, nil
		}
	}

	return 0, fmt.Errorf("no table MinerFunds")
}
