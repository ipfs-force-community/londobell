package dep

import (
	"fmt"

	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node/modules"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/beacon"
	"github.com/filecoin-project/lotus/chain/beacon/drand"
	"github.com/filecoin-project/lotus/node/modules/helpers"
)

func RandomSchedule(lc fx.Lifecycle, mctx helpers.MetricsCtx, p modules.RandomBeaconParams) (beacon.Schedule, error) {
	gen, err := p.Cs.GetGenesis(helpers.LifecycleCtx(mctx, lc))
	if err != nil {
		return nil, err
	}

	shd := beacon.Schedule{}
	for _, dc := range p.DrandConfig {
		bc, err := drand.NewDrandBeacon(gen.Timestamp, build.BlockDelaySecs, p.PubSub, dc.Config)
		if err != nil {
			return nil, fmt.Errorf("creating drand beacon: %w", err)
		}
		shd = append(shd, beacon.BeaconPoint{Start: dc.Start, Beacon: bc})
	}

	return shd, nil
}
