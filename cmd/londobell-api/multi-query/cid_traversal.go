package multiquery

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// probeLog 独立于本包既有的 log（多一个 tag 便于在聚合器主机上单独放大探测链路的日志）。
var probeLog = logging.Logger("multi-query/cid-probe")

// probeConcurrency 是探测阶段的并发度，沿用本仓既有口径（MultiPagingQuery/MultiRangeQuery 都用 16）。
const probeConcurrency = 16

// 覆盖索引探测用到的字段/集合名（避免字面量在多处重复）。
const (
	// probeFieldCid 是 ExecTrace 上的消息 cid（Cid_1 索引）。
	probeFieldCid = "Cid"
	// probeFieldSignedCid 是带签名消息落库的 cid（SignedCid_1 索引）。
	probeFieldSignedCid = "SignedCid"
	// probeProjectionID 是必须显式排除的 _id（保留默认取出路径会破坏覆盖索引）。
	probeProjectionID = "_id"
	// blockMessageTable 是需要 AllowDiskUse 的表（与 MultiTraversalQuery 口径一致）。
	blockMessageTable = "BlockMessage"
)

// CidProbeFields 是「消息 cid 落在哪个库」的探测键字段顺序。
//
// 为什么两个字段都要试：ExecTrace 上一条消息可能落在 Cid（裸消息）也可能落在 SignedCid
// （带签名消息）上，写哪一个取决于该消息是否被重新签名 —— 只探 Cid 会漏掉后一种落库形态。
// 为什么 Cid 在前：Cid_1 是主用索引（冷库 mongo08 实测 13.2GB），绝大多数消息命中它；
// SignedCid_1 只有 1.5GB，用兜底。
var CidProbeFields = []string{probeFieldCid, probeFieldSignedCid}

// cidProbeFilter 构造探测用的单键过滤器。
//
// 只允许单键：
//   - 单键才能命中 {Cid:1} / {SignedCid:1} 这类单字段索引；
//   - 一旦写成 $or:[{Cid:X},{SignedCid:X}]（等价语义的常见写法），执行计划会退化成
//     SUBPLAN + FETCH（mongo08 实测），比单键探测贵一档，且无法保持覆盖索引形态。
//
// 因此两个字段是「顺序试两次」而不是「一次 $or 试完」。
func cidProbeFilter(field, cid string) bson.D {
	return bson.D{{Key: field, Value: cid}}
}

// cidProbeProjection 构造探测用的投影：只保留 Cid、显式排除 _id。
//
// 为什么必须正好是 {_id:0, Cid:1}：
//   - Cid_1 索引的 keys={Cid:1}，投影里出现的字段全部落在索引里 ⇒ 执行计划 PROJECTION_COVERED，
//     只扫索引、不 FETCH 文档。冷库 mongo08 上用真实数据实测：keysExamined=1、docsExamined=0
//     （cid 不存在时 keysExamined=0），是能找到的最便宜的探测形态；
//   - 只要投影里出现索引外的字段（更不用说完全不投影），计划就会退化成 FETCH，必须读文档页 ——
//     冷盘（HDD 冷存储）上一次 0.8KB 的随机读就是毫秒起步，12 个库串起来就是几百毫秒；
//   - _id 必须显式排除：保留默认的 _id 取出路径同样会破坏覆盖索引（COLLSCAN/FETCH 兜底）。
func cidProbeProjection() bson.D {
	return bson.D{{Key: probeProjectionID, Value: 0}, {Key: probeFieldCid, Value: 1}}
}

// cidProbeFunc 在单个集合上按单键做一次覆盖索引探测，返回是否命中。
//
// 抽成函数类型是为了让「先探哪个字段 / 试哪些库 / 假命中后怎么走」这套编排逻辑
// 可以脱离真库做单元测试（见 cid_traversal_test.go）。
type cidProbeFunc func(ctx context.Context, col *mongo.Collection, field, cid string) (bool, error)

// cidAggregateFunc 在单个集合上执行完整聚合管道。
type cidAggregateFunc func(ctx context.Context, col *mongo.Collection, pipe interface{}, tableName string) ([]bson.M, error)

// mongoCidProbe 是 cidProbeFunc 的真实实现：单键 find + 投影 {_id:0,Cid:1} + limit(1)。
//
// limit(1) 与 FindOne 的 singleBatch 语义等价，这里显式用 Find+SetLimit(1)，
// 好处是与 explain 里验证过的「find().limit(1)」形态逐字对应，便于回归。
func mongoCidProbe(ctx context.Context, col *mongo.Collection, field, cid string) (bool, error) {
	cur, err := col.Find(ctx, cidProbeFilter(field, cid),
		options.Find().SetProjection(cidProbeProjection()).SetLimit(1))
	if err != nil {
		return false, err
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	if cur.Next(ctx) {
		return true, nil
	}

	// Next 返回 false 有两种原因：游标耗尽（= 未命中）或出错。
	// 必须把 cur.Err() 带出去，不能把「探测失败」当成「该库没有」——
	// 那会让接口静默返回空结果（比报错更难排查）。
	return false, cur.Err()
}

// mongoCidAggregate 是 cidAggregateFunc 的真实实现，与 MultiTraversalQuery 中
// 「按表名决定是否 AllowDiskUse」的处理保持一致。
func mongoCidAggregate(ctx context.Context, col *mongo.Collection, pipe interface{}, tableName string) ([]bson.M, error) {
	var (
		cur *mongo.Cursor
		err error
	)
	if tableName == blockMessageTable {
		cur, err = col.Aggregate(ctx, pipe, options.Aggregate().SetAllowDiskUse(true))
	} else {
		cur, err = col.Aggregate(ctx, pipe)
	}
	if err != nil {
		return nil, err
	}

	var res []bson.M
	if err := cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// cidCandidate 是一个「可能持有该 cid」的候选库（精确到集合）。
type cidCandidate struct {
	order     int // 在候选列表中的顺序，用于把并发探测结果还原成确定顺序
	dtype     smodel.DType
	col       *mongo.Collection
	desc      string // 仅用于日志/错误信息，例如 dp_cold_08.ExecTrace
	countList CountUtil
}

// columnDesc 生成 "库.集合" 形式的描述，便于在错误信息里一眼定位到哪个库出的问题。
func columnDesc(col *mongo.Collection) string {
	if col == nil {
		return "<nil>"
	}
	if db := col.Database(); db != nil {
		return db.Name() + "." + col.Name()
	}
	return col.Name()
}

// cidCandidates 把 countLists 展开成候选集合列表，顺序与 MultiTraversalQuery 的
// priorityLists（tmp/formal）+ delayedLists（冷库）一致：先热点库，再冷库。
//
// 与 MultiTraversalQuery 一样，缺少目标表的库直接跳过（不报错）：
// 原来 delayedLists 的 goroutine 遍历完所有集合没找到表也只是返回 nil。
func cidCandidates(countLists []CountUtil, tableName string) []cidCandidate {
	candidates := make([]cidCandidate, 0, len(countLists))

	for _, countList := range countLists {
		for _, col := range countList.Cols.Cols {
			if col == nil || col.Name() != tableName {
				continue
			}

			candidates = append(candidates, cidCandidate{
				order:     len(candidates),
				dtype:     countList.DType,
				col:       col,
				desc:      columnDesc(col),
				countList: countList,
			})

			// 一个库里同名集合取第一个，与 MultiTraversalQuery 的处理一致
			break
		}
	}

	return candidates
}

// MultiTraversalQueryByCid 是 MultiTraversalQuery 在「按消息 cid 取详情」场景下的两阶段版本。
//
// 为什么要拆成两阶段（冷库上的 explain 事实）：
//  1. 只有「按 cid 等值定位」这一小类查询能靠索引直接定库。分片粒度是「一个 DSN 一段高度区间，
//     段内所有表同库」，cid 与高度区间没有任何映射关系，所以现在只能在 12 个库上各跑一次完整
//     聚合管道，成本 100% 花在「找它在哪个库」上：实测冷盘单请求 352ms、快盘 86ms；
//  2. ExecTrace 上 Cid_1 覆盖索引探测极便宜：find({Cid:X},{_id:0,Cid:1}).limit(1) 的执行计划是
//     PROJECTION_COVERED，keysExamined=1、docsExamined=0（cid 不存在时 keysExamined=0）；
//  3. 而原来线上管道（pool-monitor/trace_for_message.js）的 $match 是
//     {$and:[{IsBlock:true},{$or:[{Cid:ctx.Cid},{SignedCid:ctx.Cid}]}]}，后面还有
//     $lookup Message + $unwind ⇒ 必然取文档，且 $or 会让计划变成 SUBPLAN + FETCH，
//     所以只能用在「已经知道库」之后；
//  4. Message 的 cid 在链上唯一 ⇒ 真正持有该 cid 的库最多只有一个，探测命中一个就够。
//
// 于是这里：先并发/顺序探测定位库（只读索引），命中后只在那个库上跑原来的完整管道；
// 探测命中而管道为空（假命中：探测看得到 Cid/SignedCid，看不到 IsBlock，IsBlock=false 的行
// 也会命中探测）时继续试下一个候选库，全部候选都空才返回空 —— 保证不出现「探测命中却查不到」。
//
// 返回值、错误语义与 MultiTraversalQuery 保持完全一致（返回管道原始 []bson.M），
// 调用方（trace_for_message 等）的响应结构与前端契约不变。
func MultiTraversalQueryByCid(ctx context.Context, pipe interface{}, countLists []CountUtil, tableName, cid string) ([]bson.M, error) {
	return traversalQueryByCid(ctx, mongoCidProbe, mongoCidAggregate, pipe, countLists, tableName, cid, CidProbeFields)
}

// traversalQueryByCid 是两阶段查询的编排本体，探测/聚合以函数注入，便于单测。
func traversalQueryByCid(ctx context.Context, probe cidProbeFunc, agg cidAggregateFunc, pipe interface{},
	countLists []CountUtil, tableName, cid string, probeFields []string) ([]bson.M, error) {

	candidates := cidCandidates(countLists, tableName)

	hot := make([]cidCandidate, 0, len(candidates))
	cold := make([]cidCandidate, 0, len(candidates))
	for _, cand := range candidates {
		if cand.dtype == smodel.Formal || cand.dtype == smodel.Tmp {
			hot = append(hot, cand)
			continue
		}
		cold = append(cold, cand)
	}

	// 第一阶段（热点库，顺序）：与 MultiTraversalQuery「优先查 tmp/formal，未查到再并发查冷库」
	// 完全一致 —— 逐个热点库「探测 → 只在该库跑管道」，一旦拿到结果立刻返回，
	// 排在后面的热点库与所有冷库连探测都不发（错误语义也因此保持一致：
	// 前一个热点库出错就直接上报，后面的库不再触碰）。
	for _, cand := range hot {
		hit, err := probeCandidate(ctx, probe, cand, cid, probeFields)
		if err != nil {
			return nil, err
		}
		if !hit {
			continue
		}

		res, err := aggregateOnHits(ctx, agg, []cidCandidate{cand}, pipe, tableName)
		if err != nil {
			return nil, err
		}
		if len(res) > 0 {
			return res, nil
		}
	}

	// 第一阶段（冷库，并发）：冷盘上每个库的探测只是一次覆盖索引 seek，
	// 并发发出，整个探测阶段的耗时约等于最慢的那一个库（原 MultiTraversalQuery
	// 对冷库也是并发扇出，并发度沿用同一口径）。
	coldHits, err := probeCandidates(ctx, probe, cold, cid, probeFields)
	if err != nil {
		return nil, err
	}

	// 第二阶段：只在命中的库里跑完整管道；命中但管道为空则继续试下一个候选库。
	return aggregateOnHits(ctx, agg, coldHits, pipe, tableName)
}

// probeCandidate 顺序探测单个候选库：按 probeFields 顺序（Cid → SignedCid）试，命中即停。
//
// 同一条消息在一个库里不会同时以 Cid/SignedCid 落两行，多探一次只有成本没有收益。
func probeCandidate(ctx context.Context, probe cidProbeFunc, cand cidCandidate, cid string, probeFields []string) (bool, error) {
	for _, field := range probeFields {
		hit, err := probe(ctx, cand.col, field, cid)
		if err != nil {
			// 探测出错必须上报：错误 ≠ 未命中。吞掉的话「库不可用」会被当成
			// 「cid 不存在」，接口静默返回空，比直接报错更难排查。
			return false, fmt.Errorf("probe %v.%v for cid %v failed: %w", cand.desc, field, cid, err)
		}

		if hit {
			probeLog.Debugf("cid %v probed to %v via %v", cid, cand.desc, field)
			return true, nil
		}
	}

	return false, nil
}

// probeCandidates 并发探测候选库，返回命中的候选（按候选顺序排列）。
//
// 库内探测复用 probeCandidate（Cid → SignedCid，命中即停）。
func probeCandidates(ctx context.Context, probe cidProbeFunc, candidates []cidCandidate, cid string, probeFields []string) ([]cidCandidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	var (
		ewg  multierror.Group
		mu   sync.Mutex
		hits = make([]cidCandidate, 0, len(candidates))
	)

	lim := limiter.New(probeConcurrency)

	for i := range candidates {
		cand := candidates[i]

		ewg.Go(func() error {
			if !lim.Acquire(ctx) {
				// ctx 已取消：与原 MultiTraversalQuery 的 limiter 处理一致，不额外报错
				return nil
			}
			defer func() {
				lim.Release(ctx)
			}()

			hit, err := probeCandidate(ctx, probe, cand, cid, probeFields)
			if err != nil {
				return err
			}

			if hit {
				mu.Lock()
				hits = append(hits, cand)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return nil, err
	}

	// 并发探测的完成顺序不确定，必须按候选顺序还原：
	// 「假命中时先试哪个库」要可重复，否则同一个请求的结果会随调度漂移。
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].order < hits[j].order
	})

	if len(hits) > 0 {
		descs := make([]string, 0, len(hits))
		for _, hit := range hits {
			descs = append(descs, hit.desc)
		}
		probeLog.Debugf("cid %v hit %d/%d candidates: %v", cid, len(hits), len(candidates), descs)
	}

	return hits, nil
}

// aggregateOnHits 只在探测命中的库上跑完整聚合管道，返回第一个非空结果。
//
// 探测命中 ≠ 管道有结果：探测只看得到 Cid/SignedCid，看不到 IsBlock，
// 所以「IsBlock=false 的那一行」也会被探测命中，但它会被管道里的
// {$and:[{IsBlock:true}, ...]} 过滤掉。此时必须继续试下一个候选库 ——
// 生产端保证的只是「该 cid 至多在一个库里存在」，不是「探测命中的那一行就是管道要的行」。
func aggregateOnHits(ctx context.Context, agg cidAggregateFunc, hits []cidCandidate, pipe interface{}, tableName string) ([]bson.M, error) {
	for _, cand := range hits {
		res, err := agg(ctx, cand.col, pipe, tableName)
		if err != nil {
			// 命中库的管道报错不能被当成「这个库里没有」，原样上报
			// （与 MultiTraversalQuery 对聚合错误的处理一致）。
			return nil, fmt.Errorf("aggregate %v failed: %w", cand.desc, err)
		}

		if len(res) > 0 {
			probeLog.Debugf("cid found in %v, %d docs", cand.desc, len(res))
			return res, nil
		}

		probeLog.Debugf("probe hit %v but pipeline returned empty, trying next candidate", cand.desc)
	}

	// 全部候选都空：返回空（非 nil）结果，语义与原实现一致
	return []bson.M{}, nil
}
