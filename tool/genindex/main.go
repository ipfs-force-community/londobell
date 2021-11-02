package main

import (
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/common"
	_ "github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate"
	_ "github.com/ipfs-force-community/londobell/racailum/segment/extract/tipset"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func main() {
	fieldMap := map[string][][]string{}
	cols := []string{}

	models := schema.Models()
	for _, m := range models {
		col := m.D.CollectionName()
		if _, has := fieldMap[col]; has {
			continue
		}

		var indexes [][]string

		if midx, ok := m.D.(common.Indexed); ok {
			indexes = midx.Indexes()
		} else {
			ef := m.D.EpochField()

			if ef == nil {
				continue
			}

			if f := *ef; f != "_id" {
				indexes = [][]string{[]string{f}}
			}
		}

		fieldMap[col] = indexes
		cols = append(cols, col)
	}

	sort.Strings(cols)

	for _, col := range cols {
		indexes := fieldMap[col]
		if len(indexes) == 0 {
			fmt.Printf("// no indexes for %s\n\n", col)
			continue
		}

		for _, index := range indexes {
			if len(index) == 0 {
				continue
			}

			idxDoc := make(bson.D, 0, len(index))

			for ii := range index {
				fname := index[ii]
				val := 1
				if strings.HasPrefix(fname, "-") {
					fname = fname[1:]
					val = -1
				}

				fname = strings.TrimSpace(fname)
				if fname == "" {
					continue
				}

				idxDoc = append(idxDoc, bson.E{
					Key:   fname,
					Value: val,
				})
			}

			if len(idxDoc) == 0 {
				continue
			}

			b, err := bson.MarshalExtJSON(idxDoc, false, false)
			if err != nil {
				panic(fmt.Errorf("marshal json for index document: %w", err))
			}

			fmt.Printf("db.%s.createIndex(%s, {\"sparse\": true});\n", col, string(b))
		}

		fmt.Println()
	}
}
