package tipset

import (
	"context"

	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"

	"github.com/ipfs-force-community/londobell/common"
)

func init() {
	sched := filcns.DefaultUpgradeSchedule()
	for si := range sched {
		if sched[si].Expensive {
			expensiveNetworkVersions[sched[si].Network] = struct{}{}
		}
	}
}

var expensiveNetworkVersions = map[network.Version]struct{}{}

func IsExpensive(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet) bool {
	prev := stm.GetNetworkVersion(ctx, ts.Parent.Height())
	next := stm.GetNetworkVersion(ctx, ts.Height())
	if prev == next {
		return false
	}

	_, is := expensiveNetworkVersions[next]
	return is
}
