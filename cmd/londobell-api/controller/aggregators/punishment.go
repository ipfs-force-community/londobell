package aggregators

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetPunishment(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetPunishment")
	req := model.CommonReq{}
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
	alog.Info("get punishment")
	punishmentRes, err := getPunishment(ctx, req, &countUtils)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}
	res.Data = punishmentRes
	c.JSON(http.StatusOK, res)
}

func getPunishment(ctx context.Context, req model.CommonReq, countUtils *[]multiquery.CountUtil) ([]model.PunishmentRes, error) {

	alog := log.With("method", "GetPunishment")
	var (
		punishmentRes []model.PunishmentRes
	)
	// get target ExecTrace

	query := []byte(fmt.Sprintf(`
		[
			{
				$match: {
					"Msg.To": "099",
					"Msg.Method": 0,
					"SubCallCount": 0,
					"Detail.Return": null,
					"GasCost": null,
					"Epoch": {
						$eq: %d,
					}
				}
			}  
		]	
			`, req.StartEpoch))
	queryRes, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, *countUtils, query, req, "ExecTrace")
	if err != nil {
		return nil, err
	}

	if len(queryRes) == 0 {
		return punishmentRes, nil
	}
	for index := range queryRes {
		var id string
		var ok bool
		if id, ok = queryRes[index]["_id"].(string); !ok {
			break
		}
		parentID, err := getParentID(id)
		if err != nil {
			break
		}
		query := []byte(fmt.Sprintf(`
		[
			{
				$match: {
					"_id": "%s",
				}
			},
			{
				$lookup: {
					from: "Message",
					let: {
						id: "$Cid",
					},
					pipeline: [
						{
							$match: {
								$expr: {
									$and: [
										{$eq: ["$_id", "$$id"]},
									   {$or: [
											   {$eq: ["$Detail.Method", "TerminateSectors"]},
											   {$eq: ["$Detail.Method", "OnDeferredCronEvent"]},
											   {$eq: ["$Detail.Method", "ApplyRewards"]},
											   {$eq: ["$Detail.Method", "DeclareFaultsRecovered"]}
										   ]
									   }                                
									]
								}
							}
						}
					],
					as: "message"
				}
			},
            {
    
				$project: {
						_id: 0,
						Methods: "$message.Detail.Method"                
					}
			}

		]	
			`, parentID))

		parent, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, *countUtils, query, req, "ExecTrace")
		if err != nil {
			return nil, err
		}

		if len(parent) == 0 {
			return punishmentRes, nil
		}

		var method string

		if methods, ok := parent[0]["Methods"].(primitive.A); !ok {
			alog.Info(parent)
			break
		} else {
			if len(methods) == 0 {
				break
			}
			if method, ok = methods[0].(string); !ok {
				break
			}
		}
		penaltyType := "sector"
		if method == "ApplyRewards" {
			penaltyType = "block"
		}

		query = []byte(fmt.Sprintf(`
			[
				{
					$match: {
						"_id": "%s",
					}
				},
				{
					$lookup: {
						from: "Message",
						let: {
							id: "$Cid",
						},
						pipeline: [
							{
								$match: {
									$expr: {
										$and: [
											{$eq: ["$_id", "$$id"]},
										]
									}
								}
							}
						],
						as: "burnMessage"
					}
				},
				{
					$unwind: "$burnMessage"
				},
				{
					$project: {
						_id: 0,
						Miner: "$burnMessage.From",
						Epoch: "$Epoch",
						BlockTime: {
							$toDate: {$add: [{$toDecimal: {
										$dateFromString: {
											dateString: "2020-08-25T06:00:00", //格式："2020-08-25T06:00:00"
											timezone: "Asia/Shanghai"
										}
									}}, {$multiply: ["$Epoch", 30*1000]}]}
						},
						Value: "$burnMessage.Value",
						PenaltyType: "%s",
						Source: "londobell"
					}
				}          
			]
			`, id, penaltyType))
		punishment, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, *countUtils, query, req, "ExecTrace")
		if err != nil {
			return nil, err
		}
		if len(punishment) == 0 {
			return punishmentRes, nil
		}

		raw := punishment
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var res []model.PunishmentRes
		err = json.Unmarshal(rawByte, &res)
		if err != nil {
			return nil, err
		}
		punishmentRes = append(punishmentRes, res...)

	}

	// get message

	return punishmentRes, nil
}

func getParentID(id string) (string, error) {
	elems := strings.Split(id, "-")
	if len(elems) < 2 {
		return id, errors.New("invalid id: " + id)
	}
	id = strings.TrimSuffix(id, "-"+elems[len(elems)-1])
	return id, nil
}
