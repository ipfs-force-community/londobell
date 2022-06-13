package dep

import (
	"bytes"
	"fmt"

	"github.com/ipld/go-car"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
)

func LoadGenesis(genBytes []byte) func(fx.Lifecycle, helpers.MetricsCtx, dtypes.ChainBlockstore) modules.Genesis {
	return func(lc fx.Lifecycle, mctx helpers.MetricsCtx, bs dtypes.ChainBlockstore) modules.Genesis {
		return func() (header *types.BlockHeader, e error) {
			ctx := helpers.LifecycleCtx(mctx, lc)
			c, err := car.LoadCar(ctx, bs, bytes.NewReader(genBytes))
			if err != nil {
				return nil, fmt.Errorf("loading genesis car file failed: %w", err)
			}
			if len(c.Roots) != 1 {
				return nil, xerrors.New("expected genesis file to have one root")
			}
			root, err := bs.Get(ctx, c.Roots[0])
			if err != nil {
				return nil, err
			}

			h, err := types.DecodeBlock(root.RawData())
			if err != nil {
				return nil, fmt.Errorf("decoding block failed: %w", err)
			}
			return h, nil
		}
	}
}
