package segment

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/dtynn/londobell/common"
)

func (s *Segment) insertMany(ctx context.Context, l *zap.SugaredLogger, docSets [][]common.Document) error {
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

		insertOps++
		inserted, err := s.db.Insert(ctx, col, docs[col])
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
