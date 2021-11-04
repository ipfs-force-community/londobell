package main

import (
	"fmt"
	"log"
	"os"

	"github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/lotus/chain/actors"
	exported0 "github.com/filecoin-project/specs-actors/actors/builtin/exported"
	exported2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	exported3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/exported"
	exported4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/exported"
	exported5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/exported"
	exported6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/exported"

	"github.com/ipfs-force-community/londobell/tool/actordiff/internal"
)

func main() {
	exports := [][]rt.VMActor{
		exported0.BuiltinActors(),
		exported2.BuiltinActors(),
		exported3.BuiltinActors(),
		exported4.BuiltinActors(),
		exported5.BuiltinActors(),
		exported6.BuiltinActors(),
	}

	if len(exports) != len(actors.Versions) {
		log.Fatalf("length of exports & versions are not equal, %d != %d", len(exports), len(actors.Versions))
	}

	specs := make([]*internal.Specs, 0, len(exports))
	for i := range exports {
		spec, err := internal.ResolveSpecs(actors.Versions[i], exports[i])
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
		fmt.Fprintln(os.Stdout, "")
	}
}
