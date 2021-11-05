package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/filecoin-project/lotus/chain/actors"

	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/tool/actordiff/internal"
)

func main() {
	specs := make([]*internal.Specs, 0, len(actor.Specs))
	for i := range actor.Specs {
		spec, err := internal.ResolveSpecs(actors.Versions[i], actor.Specs[i])
		if err != nil {
			log.Fatalf("resolve specs v%d: %s", actors.Versions[i], err)
		}

		specs = append(specs, spec)
	}

	for i := 1; i < len(specs); i++ {
		diff := internal.CompareSpecs(specs[i-1], specs[i])
		fmt.Fprintf(os.Stdout, "Specs v%d => v%d:\n", specs[i-1].Ver, specs[i].Ver)
		for _, add := range diff.Adds {
			fmt.Fprintf(os.Stdout, "\t+ %s\n", add.String())
		}

		for _, minus := range diff.Minuses {
			fmt.Fprintf(os.Stdout, "\t- %s\n", minus.String())
		}

		for _, schange := range diff.StateChanges {
			fmt.Fprintf(os.Stdout, "\t> %s\n", schange.NextActor.String())

			for _, fadd := range schange.Adds {
				fmt.Fprintf(os.Stdout, "\t\t+ %s: %s\n", fadd.Name, fadd.Type)
			}

			for _, fminus := range schange.Minuses {
				fmt.Fprintf(os.Stdout, "\t\t- %s: %s\n", fminus.Name, fminus.Type)
			}

			for _, fchanges := range schange.Changes {
				fmt.Fprintf(os.Stdout, "\t\t> %s\n", fchanges[0].Name)
				fmt.Fprintf(os.Stdout, "\t\t\t %s => %s\n", fchanges[0].Type, fchanges[1].Type)
			}
		}

		oldWatchOuts := actor.WatchOuts[i-1]
		newWatchOuts := actor.WatchOuts[i]
		for wi := range oldWatchOuts {
			oldType := reflect.TypeOf(oldWatchOuts[wi])
			newType := reflect.TypeOf(newWatchOuts[wi])
			typeDiff := internal.CompareStructs(oldType, newType)
			if !typeDiff.IsEmpty() {
				fmt.Fprintf(os.Stdout, "\t> %s\n", newType.String())

				for _, fadd := range typeDiff.Adds {
					fmt.Fprintf(os.Stdout, "\t\t+ %s: %s\n", fadd.Name, fadd.Type)
				}

				for _, fminus := range typeDiff.Minuses {
					fmt.Fprintf(os.Stdout, "\t\t- %s: %s\n", fminus.Name, fminus.Type)
				}

				for _, fchanges := range typeDiff.Changes {
					fmt.Fprintf(os.Stdout, "\t\t> %s\n", fchanges[0].Name)
					fmt.Fprintf(os.Stdout, "\t\t\t %s => %s\n", fchanges[0].Type, fchanges[1].Type)
				}
			}
		}

		fmt.Fprintln(os.Stdout, "")
	}
}
