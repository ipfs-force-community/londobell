package segment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
)

func (s *Segment) insertMany(ctx context.Context, l *zap.SugaredLogger, docSets [][]common.Document) error {
	if len(docSets) == 0 {
		return nil
	}

	limit := s.opts.Persist.BatchInsertLimit
	insertDocs := map[string][]interface{}{}
	updateDocs := map[string][]interface{}{}
	insertedCounts := map[string]int{}
	matchedCounts := map[string]int{}
	modifiedCounts := map[string]int{}
	upsertedCounts := map[string]int{}
	totals := map[string]int{}
	insertOps := 0
	updateOps := 0

	insert := func(col string) error {
		if len(insertDocs[col]) == 0 {
			return nil
		}
		var (
			inserted int
			err      error
		)

		insertOps++
		inserted, err = s.db.Insert(ctx, col, insertDocs[col])
		if err != nil && strings.Contains(err.Error(), "too large") {
			l.Warnw("batch insert contains oversized documents, retrying individually", "collection", col)
			var truncated int
			inserted = 0
			for _, doc := range insertDocs[col] {
				n, e := s.db.Insert(ctx, col, []interface{}{doc})
				if e != nil {
					if strings.Contains(e.Error(), "too large") {
						shrunk, shrinkErr := truncateOversizedDoc(doc)
						if shrinkErr != nil {
							l.Warnw("unable to truncate oversized document", "collection", col, "error", shrinkErr)
							truncated++
							continue
						}
						n, e = s.db.Insert(ctx, col, []interface{}{shrunk})
						if e != nil {
							l.Warnw("still too large after truncation", "collection", col, "error", e)
							truncated++
							continue
						}
						truncated++
					} else {
						return e
					}
				}
				inserted += n
			}
			if truncated > 0 {
				l.Warnw("oversized documents truncated to allow sync to continue", "collection", col, "truncated", truncated, "inserted", inserted)
			}
		} else if err != nil {
			return err
		}

		insertDocs[col] = insertDocs[col][:0]
		insertedCounts[col] = insertedCounts[col] + inserted
		return nil
	}

	update := func(col string) error {
		if len(updateDocs[col]) == 0 {
			return nil
		}

		updateOps++
		for _, doc := range updateDocs[col] {
			// update操作不允许在一个库中重新跑之前epoch，会覆盖后面epoch的记录
			primaryKeyValue, err := ExtractPrimaryKeyValue(doc)
			if err != nil {
				return err
			}
			epochValue, err := ExtractEpochValue(doc)
			if err != nil {
				return err
			}

			updateDoc, err := ExtractUpdateDoc(doc)
			if err != nil {
				return err
			}

			var existingDoc bson.M
			err = s.db.FindOne(ctx, col, bson.M{"_id": primaryKeyValue}).Decode(&existingDoc)
			if err != nil && err != mongo.ErrNoDocuments {
				return err
			}

			var res = &mongo.UpdateResult{}
			if err == mongo.ErrNoDocuments {
				res, err = s.db.Update(ctx, col, bson.M{"_id": primaryKeyValue}, bson.M{"$set": updateDoc})
				if err != nil {
					return err
				}
			} else {
				existingEpoch, ok := existingDoc["Epoch"].(int64)
				if !ok {
					return fmt.Errorf("field Epoch is not int64 type: %+v", existingDoc)
				}

				if epochValue >= existingEpoch {
					res, err = s.db.Update(ctx, col, bson.M{"_id": primaryKeyValue}, bson.M{"$set": updateDoc})
					if err != nil {
						return err
					}
				} else {
					continue
				}
			}

			matchedCounts[col] = matchedCounts[col] + int(res.MatchedCount)
			modifiedCounts[col] = modifiedCounts[col] + int(res.ModifiedCount)
			upsertedCounts[col] = upsertedCounts[col] + int(res.UpsertedCount)
		}

		updateDocs[col] = updateDocs[col][:0]
		return nil
	}

	for si := range docSets {
		for di := range docSets[si] {
			d := docSets[si][di]
			colName := d.CollectionName()
			if !d.IsMutable() {
				insertDocs[colName] = append(insertDocs[colName], d)
			} else {
				updateDocs[colName] = append(updateDocs[colName], d)
			}

			totals[colName] = totals[colName] + 1

			if len(insertDocs[colName]) >= limit {
				if err := insert(colName); err != nil {
					return err
				}
			}

			if len(updateDocs[colName]) >= limit {
				if err := update(colName); err != nil {
					return err
				}
			}
		}
	}

	for col := range insertDocs {
		if err := insert(col); err != nil {
			return err
		}
	}

	for col := range updateDocs {
		if err := update(col); err != nil {
			return err
		}
	}

	insertColNames := make([]string, 0, len(insertDocs))
	updateColNames := make([]string, 0, len(updateDocs))
	for col := range insertDocs {
		insertColNames = append(insertColNames, col)
	}
	for col := range updateDocs {
		updateColNames = append(updateColNames, col)
	}

	sort.Strings(insertColNames)
	sort.Strings(updateColNames)

	logFields := make([]interface{}, 0, (len(insertColNames)+len(updateColNames))*2+2)
	logFields = append(logFields, "insertOps", insertOps, "updateOps", updateOps)
	for _, col := range insertColNames {
		logFields = append(logFields, col, fmt.Sprintf("%d/%d", insertedCounts[col], totals[col]))
	}

	for _, col := range updateColNames {
		logFields = append(logFields, col, fmt.Sprintf("matched: %v, modified: %d, upserted: %d, total: %d", matchedCounts[col], modifiedCounts[col], upsertedCounts[col], totals[col]))
	}

	l.Infow("documents inserted and updated", logFields...)

	return nil
}

const maxBSONDocumentSize = 15 * 1024 * 1024
const maxFieldSize = 10 * 1024 * 1024

func truncateOversizedDoc(doc interface{}) (interface{}, error) {
	raw, err := bson.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal doc: %w", err)
	}

	if len(raw) <= maxBSONDocumentSize {
		return doc, nil
	}

	var m bson.M
	if err := bson.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal to bson.M: %w", err)
	}

	truncated := truncateLargeFields(m)

	truncated["TruncatedFields"] = true
	return truncated, nil
}

func truncateLargeFields(m bson.M) bson.M {
	for key, val := range m {
		switch v := val.(type) {
		case []byte:
			if len(v) > maxFieldSize {
				m[key] = append(v[:maxFieldSize], []byte("...<truncated>")...)
			}
		case string:
			if len(v) > maxFieldSize {
				m[key] = v[:maxFieldSize] + "...<truncated>"
			}
		case bson.M:
			m[key] = truncateLargeFields(v)
		case bson.A:
			m[key] = truncateLargeArray(v)
		}
	}
	return m
}

func truncateLargeArray(a bson.A) bson.A {
	for i, val := range a {
		switch v := val.(type) {
		case []byte:
			if len(v) > maxFieldSize {
				a[i] = append(v[:maxFieldSize], []byte("...<truncated>")...)
			}
		case string:
			if len(v) > maxFieldSize {
				a[i] = v[:maxFieldSize] + "...<truncated>"
			}
		case bson.M:
			a[i] = truncateLargeFields(v)
		case bson.A:
			a[i] = truncateLargeArray(v)
		}
	}
	return a
}

func (s *Segment) GetTipSetItemsCount(ctx context.Context, l *zap.SugaredLogger) (uint, error) {
	type ItemsCount struct {
		Counts uint
	}

	countStage := bson.D{
		{
			Key: "$count", Value: "Counts",
		},
	}

	var itemsCount []ItemsCount
	err := s.rdb.Aggregate(ctx, "Tipset", mongo.Pipeline{countStage}, &itemsCount)
	if err != nil {
		return 0, err
	}

	if len(itemsCount) == 0 {
		l.Infow("GetTipSetItemsCount", "temporary db has no item", len(itemsCount))
		return 0, nil
	}
	if len(itemsCount) != 1 {
		return 0, fmt.Errorf("temporary db has datas but the length of itemsCount is not equal 1, len(itemsCount): %v", len(itemsCount))
	}

	l.Infow("GetTipSetItemsCount", "itemsCount", itemsCount[0].Counts)

	return itemsCount[0].Counts, nil
}

func (s *Segment) GetLatestHeightForTipSet(ctx context.Context, l *zap.SugaredLogger) (abi.ChainEpoch, error) {
	type FirstHeight struct {
		Height abi.ChainEpoch `bson:"_id"`
	}

	var firstHeightRes []FirstHeight

	sortStage := bson.D{
		{
			Key: "$sort", Value: bson.D{
				{
					Key: "_id", Value: -1,
				},
			},
		},
	}

	limitStage := bson.D{
		{
			Key: "$limit", Value: 1,
		},
	}

	projectStage := bson.D{
		{
			Key: "$project", Value: bson.D{
				{
					Key: "_id", Value: 1,
				},
			},
		},
	}

	err := s.rdb.Aggregate(ctx, "Tipset", mongo.Pipeline{sortStage, limitStage, projectStage}, &firstHeightRes)
	if err != nil {
		return 0, err
	}

	if len(firstHeightRes) != 1 {
		return 0, fmt.Errorf("get first height for table Tipset failed, the length of firstHeightRes is %v", len(firstHeightRes))
	}

	return firstHeightRes[0].Height, nil
}

func (s *Segment) DeleteItemsByEpoch(ctx context.Context, l *zap.SugaredLogger, epoch abi.ChainEpoch, many, before bool) error {
	//ExecTrace: Epoch
	//Message: Detail.PackedHeight
	//Tipset: _id
	var (
		epochFilter  interface{}
		idFliter     interface{}
		heightFilter interface{}
		filterMap    = make(map[string]interface{})
	)
	if many {
		// delete items before epoch
		if before {
			epochFilter = bson.D{{Key: "Epoch", Value: bson.D{{Key: "$lte", Value: epoch}}}}
			idFliter = bson.D{{Key: "_id", Value: bson.D{{Key: "$lte", Value: epoch}}}}
			heightFilter = bson.D{{Key: "Detail.PackedHeight", Value: bson.D{{Key: "$lte", Value: epoch}}}}
		} else {
			// delete items after epoch
			epochFilter = bson.D{{Key: "Epoch", Value: bson.D{{Key: "$gt", Value: epoch}}}}
			idFliter = bson.D{{Key: "_id", Value: bson.D{{Key: "$gt", Value: epoch}}}}
			heightFilter = bson.D{{Key: "Detail.PackedHeight", Value: bson.D{{Key: "$gt", Value: epoch}}}}
		}
	} else {
		// only delete items at epoch
		epochFilter = bson.D{{Key: "Epoch", Value: epoch}}
		idFliter = bson.D{{Key: "_id", Value: epoch}}
		heightFilter = bson.D{{Key: "Detail.PackedHeight", Value: epoch}}
	}

	filterMap["ExecTrace"] = epochFilter
	filterMap["Message"] = heightFilter
	filterMap["Tipset"] = idFliter
	//filterMap["MessageBlock"] = epochFilter
	filterMap["BlockMessage"] = epochFilter
	filterMap["BlockHeader"] = epochFilter
	filterMap["ActorMessage"] = epochFilter
	filterMap["EthHash"] = epochFilter
	filterMap["EventsRoot"] = epochFilter
	filterMap["ExplicitMessage"] = epochFilter
	filterMap["EvmInitCode"] = epochFilter
	filterMap["ActorEvent"] = epochFilter
	filterMap["ChangedActor"] = epochFilter
	filterMap["ActorAddress"] = epochFilter
	tables := []string{"ExecTrace", "Message", "Tipset", "BlockMessage", "BlockHeader", "ActorMessage", "EthHash", "EventsRoot", "ExplicitMessage", "EvmInitCode", "ActorEvent", "ChangedActor", "ActorAddress"}

	for _, table := range tables {
		//todo: 根据表名构造出document
		deleted, err := s.db.Delete(ctx, table, filterMap[table])
		if err != nil {
			return err
		}

		l.Infow("DeleteItemsByEpoch", "table", table, "deleted", deleted, "epoch", epoch, "many", many, "before", before)
	}

	return nil
}
