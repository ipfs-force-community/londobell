package main

import (
	"fmt"
	"os"
	"sort"

	_ "github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/reg"
)

func main() {
	all := reg.All()
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	for _, one := range all {
		fmt.Fprintln(os.Stdout, one.Name)
	}
}
