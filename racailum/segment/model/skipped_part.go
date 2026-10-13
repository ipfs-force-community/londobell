package model

import (
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_ common.IndexedDocument = (*SkippedPart)(nil)

	skippedPartColName    = getColName(SkippedPart{})
	skippedPartEpochField = "from_epoch"
)

// SkippedPart 是「批抽取时被登记跳过的 part」的台账行（集合 SkippedPart，写在 bell 的写库，
// 与 FinalHeight 同库）。
//
// 去重口径：_id = "<from_epoch>-<to_epoch>"，天然对同一区间去重——同一区间重复命中时
// 只更新 last_seen / attempts，不会插新行（见 racailum/segment/ledger.go 的 upsert 写法）。
// 之所以不再另建唯一索引：复合 _id 本身就是唯一键；而且本段明确禁止在代码里建索引。
// 若运营侧仍希望有一份显式的唯一索引，可手工执行（本段不在代码里做）：
//
//	db.getCollection("SkippedPart").createIndex({from_epoch: 1, to_epoch: 1}, {unique: true})
type SkippedPart struct {
	ID         string         `bson:"_id"`         // "<from_epoch>-<to_epoch>"，天然唯一
	FromEpoch  abi.ChainEpoch `bson:"from_epoch"`  // part 起始高度
	ToEpoch    abi.ChainEpoch `bson:"to_epoch"`    // part 结束高度
	ErrorClass string         `bson:"error_class"` // state | data
	ErrDigest  string         `bson:"err_digest"`  // 错误摘要（已截断）
	CID        string         `bson:"cid"`         // 从错误正文提取到的根 CID（提不到则为空）
	Attempts   int            `bson:"attempts"`    // 重复命中次数
	FirstSeen  time.Time      `bson:"first_seen"`
	LastSeen   time.Time      `bson:"last_seen"`
	HealedAt   *time.Time     `bson:"healed_at"` // 本段留空；heal 闭环是后续段的事
}

// SkippedPartCollection 返回台账集合名。
func SkippedPartCollection() string { return skippedPartColName }

// SkippedPartID 生成台账 _id：同一 (from, to) 恒定同值，实现「只更新、不插新行」。
func SkippedPartID(from, to abi.ChainEpoch) string {
	return fmt.Sprintf("%d-%d", from, to)
}

// CollectionName impl common.Document
func (p *SkippedPart) CollectionName() string { return skippedPartColName }

// EpochField impl common.Document。
// 台账按「区间的起始高度」参与按 epoch 的筛选/重置。
func (p *SkippedPart) EpochField() *string { return &skippedPartEpochField }

// ResetPolicy impl common.Document
func (p *SkippedPart) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(skippedPartEpochField, lower, upper), true
}

// Indexes impl common.Indexed
func (p *SkippedPart) Indexes() [][]string {
	return [][]string{{"from_epoch", "to_epoch"}}
}

// IsMutable：台账行会被重复命中更新（last_seen / attempts），语义上可变。
//
// 注意：台账**不走** insertMany——insertMany 的可变分支按 Epoch 字段路由，而本表的键是
// 「区间」不是单个 Epoch。它恒由 ledger.go 的 recordSkippedPart 用「_id 过滤 + upsert」写入。
func (p *SkippedPart) IsMutable() bool { return true }
