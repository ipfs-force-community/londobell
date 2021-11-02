package grafana

import (
	"sort"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/lib/chainx"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	g, err := chainx.Genesis()
	if err != nil {
		panic(err)
	}

	genesis = g
}

var genesis *types.BlockHeader

func getCollections() []string { // nolint: deadcode
	m := schema.Models()
	colMap := map[string]struct{}{}
	for mi := range m {
		colMap[m[mi].D.CollectionName()] = struct{}{}
	}

	cols := make([]string, 0, len(colMap))
	for c := range colMap {
		cols = append(cols, c)
	}

	sort.Strings(cols)
	return cols
}

func time2epoch(t time.Time) abi.ChainEpoch {
	return abi.ChainEpoch((uint64(t.Unix()) - genesis.Timestamp) / builtin.EpochDurationSeconds)
}

func epoch2time(h abi.ChainEpoch) time.Time {
	return time.Unix(int64(genesis.Timestamp)+int64(h)*builtin.EpochDurationSeconds, 0)
}
