package main

import (
	"fmt"
	"sort"

	"github.com/dtynn/londobell/lib/mgoutil/mcodec"
	_ "github.com/dtynn/londobell/racailum/segment/extract/actorstate"
	_ "github.com/dtynn/londobell/racailum/segment/extract/tipset"
	_ "github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func main() {
	mcodec.Setup()

	models := schema.Models()
	mmap := map[string][]schema.Model{}

	cols := []string{}
	for mi := range models {

		col := models[mi].D.CollectionName()
		if _, has := mmap[col]; !has {
			cols = append(cols, col)
		}

		mmap[col] = append(mmap[col], models[mi])
	}

	sort.Strings(cols)

	fmt.Println("## Example")
	reg := mcodec.ExampleRegisry()
	for _, col := range cols {
		fmt.Printf("### %s\n", col)
		fmt.Println()

		ms := mmap[col]
		for mi := range ms {
			model := ms[mi]
			b, err := schema.FormatJSON(reg, model.D, true)
			if err != nil {
				panic(fmt.Errorf("format model schema: %w", err))
			}

			fmt.Printf("#### %s\n", model.Name)
			fmt.Println()
			fmt.Println("```")
			fmt.Println(string(b))
			fmt.Println("```")
			fmt.Println()
		}

		fmt.Println()
	}
}
