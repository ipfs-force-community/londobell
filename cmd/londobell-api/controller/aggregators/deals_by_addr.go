package aggregators

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetDealsByAddr(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetDealsByAddr")
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

	api := fullnode.API.GetAppropriateAPI()
	addrs, err := GetAllAddrs(ctx, req.Addr, api)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addrs = addrs

	var ID string
	for _, addr := range addrs {
		if addr[0] == '0' {
			ID = addr
			break
		}
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	for _, countUtil := range countUtils {
		totalCount := int64(0)
		for start := countUtil.Start; start < countUtil.End; start++ {
			// 只有零点会存
			if common.IsZeroHour(true, abi.ChainEpoch(start)) {
				count, err := GetTotalCountForDealsByAddr(ctx, cols, ID, addrs, abi.ChainEpoch(start))
				if err != nil {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}

				totalCount += count
			}
		}

		countUtil.Count = totalCount
	}

	var totalCount = int64(0)
	for _, countUtil := range countUtils {
		totalCount += countUtil.Count
	}

	sort.Slice(countUtils, func(i, j int) bool {
		return countUtils[i].Start > countUtils[j].End
	})

	length := len(countUtils)
	if length == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	startEpoch, endEpoch := countUtils[length-1].Start, countUtils[0].End

	var deals []model.Deal

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, startEpoch, endEpoch, countUtils, dealsByAddrAggregator, req, "DealProposal")
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

		err = json.Unmarshal(rawByte, &deals)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = model.DealsRes{TotalCount: totalCount, Deals: deals}
	c.JSON(http.StatusOK, res)
}

func GetTotalCountForDealsByAddr(ctx context.Context, cols multiquery.Collections, ID string, addrs []string, epoch abi.ChainEpoch) (int64, error) {
	DALock.RLock()
	if _, exist := DealsByAddrCountMap[ID]; exist {
		if count, ok := DealsByAddrCountMap[ID][epoch]; ok {
			DALock.RUnlock()
			return count, nil
		}
	}
	DALock.RUnlock()

	filter := bson.D{{Key: "Epoch", Value: bson.D{{Key: "$eq", Value: epoch}}}, {Key: "$or", Value: []bson.M{{"Client": bson.D{{Key: "$in", Value: addrs}}}, {"Provider": bson.D{{Key: "$in", Value: addrs}}}}}}

	tableName := "DealProposal"
	for _, col := range cols.Cols {
		if col != nil && col.Name() == tableName {
			count, err := col.CountDocuments(ctx, filter)
			if err != nil {
				return 0, err
			}

			DALock.Lock()
			if _, exist := DealsByAddrCountMap[ID]; !exist {
				DealsByAddrCountMap[ID] = make(map[abi.ChainEpoch]int64)
			}

			DealsByAddrCountMap[ID][epoch] = count
			DALock.Unlock()

			return count, nil
		}
	}

	return 0, fmt.Errorf("no table DealProposal")
}
