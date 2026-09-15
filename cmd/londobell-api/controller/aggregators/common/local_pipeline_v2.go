package common

import (
	"os"
	"strings"
)

// LocalPipelineV2Env 控制是否启用本仓覆盖的 aggregator pipeline：默认启用，
// 置 0/false/off/no/disable 时回退上游 londobell-aggregators 版本（灰度与秒级回滚用）。
//
// 本文件只覆盖 blocks_for_message，原因（2026-09-15 生产实测）：
//
// blocks_for_message 第二阶段的 $lookup BlockHeader 用 {$expr: {$in: ["$_id", "$$cids"]}}，
// 外侧值是 $group 出来的数组 let 变量 ⇒ MongoDB 无法为此用 _id_ 索引 ⇒ 每次请求都在目标库上全扫 BlockHeader。
// 生产实测（冷库 172.31.44.225，172 万条 / 612MB）：上游写法 3886ms，本仓写法 5ms，结果条数一致。
// 按 3.25 req/s 计，这一项此前给 11 台冷库白白压上约 2GB/s 的读需求。
// 改法：换成等值 join（localField/foreignField，走 BlockHeader._id 的 _id_ 索引），谓词不变、只换执行方式。
// 渲染走生产同一条链路（otto + ctx 注入），过滤条件与输出投影跟上游逐字比对，见 local_pipeline_v2_test.go。
//
// child_transfers_for_message 的同型写法（$lookup 里用 $indexOfBytes 做 _id 前缀匹配）不需要改：
// 那条 $expr 里同时有 {$eq: ["$Epoch", "$$epoch"]}，MongoDB 4.4 上已能用 Epoch_1 索引先定位同 epoch 的
// 少量文档，前缀判断只在这批文档上做内存过滤（实测 6 个真实 parent：4~12ms）。
// 改成 Epoch 等值 join 反而要把整个 epoch（实测 1061 条）的文档搬进聚合再 $filter，属负优化，故保持上游写法，
// 并由 TestChildTransfersStaysUpstream 钉住这个结论。
//
// 上游 pipeline 仍在 monitor 包（外部 module）；稳态后应把这条改动回灌上游仓，再删除本文件。
const LocalPipelineV2Env = "LONDOBELL_AGG_PIPELINE_V2"

// blocksForMessageAggregatorV2 与上游 blocks_for_message.js 等价，
// 仅把 $lookup 的 {$expr: {$in: ["$_id", "$$cids"]}} 换成 localField/foreignField 等值 join。
var blocksForMessageAggregatorV2 = `
[
    {
        $match: {
            Epoch: {$eq: ctx.StartEpoch},
            Messages: {$in: [ctx.Cid]}
        }
    },
    {
        $group: {
            _id: 0,
            Blocks: {$addToSet: "$_id"}
        }
    },
    {
        $lookup: {
            from: "BlockHeader",
            localField: "Blocks",
            foreignField: "_id",
            as: "blockheader"
        }
    },
    {
        $unwind: "$blockheader"
    },
    {
        $project: {
            _id: "$blockheader._id",
            Miner: "$blockheader.Miner",
            Epoch: "$blockheader.Epoch",
            Messages: "$blockheader.Messages",
            ElectionProof: "$blockheader.ElectionProof",
            Ticket: "$blockheader.Ticket",
            MessageCount: "$blockheader.MessageCount"
        }
    }
]
`

// pipelineV2Enabled 读环境变量决定是否启用本仓覆盖版本；默认启用。
// 每次调用都读环境变量，便于测试与运维用配置切换。
func pipelineV2Enabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(LocalPipelineV2Env))) {
	case "0", "false", "off", "no", "disable", "disabled":
		return false
	}
	return true
}

// BlocksForMessagePipeline 在启用覆盖时返回本仓修好的 pipeline，否则原样返回上游版本。
func BlocksForMessagePipeline(upstream []byte) []byte {
	if pipelineV2Enabled() {
		return []byte(blocksForMessageAggregatorV2)
	}
	return upstream
}
