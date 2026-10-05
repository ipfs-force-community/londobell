package aggregators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

// rewardActorAddr 是奖励 actor 的裸 ID 串（不带网络前缀）。
//
// ActorState 快照里 rewards actor 的 Addr 就存 "02"，且 common.GetIDByAddr 会拼
// buildnet.NetPrefix —— 传 "f02" 会得到 unknown address protocol，故这里直接用裸串。
const rewardActorAddr = "02"

// maxRewardStreamsRows 是本接口的单次结果行数硬上限：超过即显式报错，绝不硬扫全表。
// 粒度是整小时（Epoch % 120 == 0），2000 行约等于 83 天，远超统计页实际窗口。
const maxRewardStreamsRows = 2000

// rewardStreamsPipelineLimit 比硬上限多取 1 行，用于把「正好超限」与「刚好满」区分开：
// 取回 2001 行即说明还有更多，据此报错，而不是静默截断。
const rewardStreamsPipelineLimit = maxRewardStreamsRows + 1

// rewardStreamsAggregator 是本仓自带的 reward_streams 聚合管道（外部模块
// github.com/ipfs-force-community/londobell-aggregators 持有的现成管道不动）。
//
// 只做两件事：
//  1. $match 带上 Epoch 区间（走 (Epoch, Code, Addr) 索引前缀）与 Addr；禁止只按 Addr/Code 匹配。
//  2. $project 只取 Epoch 与 Detail 里那四个计数器（点号投影），
//     **不吐出整个 Detail** —— 里面有无用的超大平滑估计串（ThisEpochRewardSmoothed）。
//
// $sort 保证按 Epoch 升序（多库合并后顺序不保证），$limit 把物化行数钉死在硬上限 +1。
var rewardStreamsAggregator = []byte(`
[
    {
        $match: {
            Epoch: {$gte: ctx.StartEpoch, $lt: ctx.EndEpoch},
            Addr: ctx.Addr
        }
    },
    {
        $project: {
            _id: 0,
            Epoch: 1,
            "Detail.TotalStoragePowerReward": 1,
            "Detail.TotalMintedReward": 1,
            "Detail.TotalBurnMinted": 1,
            "Detail.TotalExplicitMinted": 1
        }
    },
    {
        $sort: {Epoch: 1}
    },
    {
        $limit: 2001
    }
]
`)

// rewardStreamRow 是 $project 之后的原始行：Epoch + 只含四个被投影计数器的 Detail。
type rewardStreamRow struct {
	Epoch  int64                  `json:"Epoch"`
	Detail map[string]interface{} `json:"Detail"`
}

// counterValue 取 Detail 里的计数器，缺失/空值/非字符串统一归一成 "0"，
// 保证四个字段恒定存在（调用方按固定结构解码）。
func counterValue(detail map[string]interface{}, field string) string {
	v, ok := detail[field]
	if !ok || v == nil {
		return "0"
	}

	switch s := v.(type) {
	case string:
		if s == "" {
			return "0"
		}
		return s
	default:
		return fmt.Sprintf("%v", s)
	}
}

// buildRewardStreams 把原始行归一成契约 A 的响应行，并按 Epoch 升序排序。
// 不做前值填充，只返回实际存在的快照。
func buildRewardStreams(rows []rewardStreamRow) []model.RewardStreamPoint {
	out := make([]model.RewardStreamPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.RewardStreamPoint{
			Epoch:                   r.Epoch,
			TotalStoragePowerReward: counterValue(r.Detail, "TotalStoragePowerReward"),
			TotalMintedReward:       counterValue(r.Detail, "TotalMintedReward"),
			TotalBurnMinted:         counterValue(r.Detail, "TotalBurnMinted"),
			TotalExplicitMinted:     counterValue(r.Detail, "TotalExplicitMinted"),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Epoch < out[j].Epoch })
	return out
}

// checkRewardStreamsRows 是行数硬上限的判据：超出即返回可读错误（不返回部分结果）。
func checkRewardStreamsRows(n int) error {
	if n > maxRewardStreamsRows {
		return fmt.Errorf("reward_streams: %d rows exceed hard limit %d; narrow the epoch range", n, maxRewardStreamsRows)
	}
	return nil
}

// GetRewardStreams 返回奖励 actor（f02）在 [start, end) 内的小时快照序列，
// 每行四个原始计数器（字符串，缺则 "0"）+ Epoch，按 Epoch 升序，不做前值填充。
func GetRewardStreams(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetRewardStreams")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// 空区间（含非法区间）直接返回空序列，不触达 mongo。
	if req.EndEpoch <= req.StartEpoch {
		res.Data = []model.RewardStreamPoint{}
		c.JSON(http.StatusOK, res)
		return
	}

	req.Addr = rewardActorAddr

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, common.GetCurEpoch())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var points []model.RewardStreamPoint

	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, req.StartEpoch, req.EndEpoch, countUtils, rewardStreamsAggregator, req, "ActorState")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		// 管道已 $limit 2001：命中即说明结果被截断，必须显式失败而不是返回部分数据。
		if err = checkRewardStreamsRows(len(multiResult)); err != nil {
			alog.Error(err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusInternalServerError, res)
			return
		}

		if len(multiResult) == 0 {
			res.Data = []model.RewardStreamPoint{}
			c.JSON(http.StatusOK, res)
			return
		}

		rawByte, err := json.Marshal(multiResult)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		var rows []rewardStreamRow
		if err = json.Unmarshal(rawByte, &rows); err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		points = buildRewardStreams(rows)
	}

	if points == nil {
		points = []model.RewardStreamPoint{}
	}

	res.Data = points
	c.JSON(http.StatusOK, res)
}
