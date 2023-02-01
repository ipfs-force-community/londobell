package segment

import (
	"context"
	"fmt"
	"sort"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
)

func (s *Segment) insertMany(ctx context.Context, l *zap.SugaredLogger, docSets [][]common.Document, upsert bool) error {
	if len(docSets) == 0 {
		return nil
	}

	limit := s.opts.Persist.BatchInsertLimit
	docs := map[string][]interface{}{}
	counts := map[string]int{}
	totals := map[string]int{}
	insertOps := 0

	insert := func(col string) error {
		if len(docs[col]) == 0 {
			return nil
		}
		var (
			inserted int
			err      error
		)

		insertOps++
		inserted, err = s.db.Insert(ctx, col, docs[col])
		if err != nil {
			return err
		}

		docs[col] = docs[col][:0]
		counts[col] = counts[col] + inserted
		return nil
	}

	for si := range docSets {
		for di := range docSets[si] {
			d := docSets[si][di]
			colName := d.CollectionName()
			docs[colName] = append(docs[colName], d)
			totals[colName] = totals[colName] + 1

			if len(docs[colName]) >= limit {
				if err := insert(colName); err != nil {
					return err
				}
			}
		}
	}

	for col := range docs {
		if err := insert(col); err != nil {
			return err
		}
	}

	colNames := make([]string, 0, len(docs))
	for col := range docs {
		colNames = append(colNames, col)
	}

	sort.Strings(colNames)

	logFields := make([]interface{}, 0, len(colNames)*2+2)
	logFields = append(logFields, "ops", insertOps)
	for _, col := range colNames {
		logFields = append(logFields, col, fmt.Sprintf("%d/%d", counts[col], totals[col]))
	}

	l.Infow("documents inserted", logFields...)

	return nil
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
	tables := []string{"ExecTrace", "Message", "Tipset", "BlockMessage", "BlockHeader", "ActorMessage", "EthHash", "EventsRoot", "ExplicitMessage"}

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
