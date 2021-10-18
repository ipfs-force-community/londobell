package tipset

import (
	"context"

	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"

	"github.com/dtynn/londobell/common"
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

func isExpensive(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet) bool {
	prev := stm.GetNtwkVersion(ctx, ts.Parent.Height())
	next := stm.GetNtwkVersion(ctx, ts.Height())
	if prev == next {
		return false
	}

	_, is := expensiveNetworkVersions[next]
	return is
}
