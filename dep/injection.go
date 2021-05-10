package dep

import (
	"context"

	metricsi "github.com/ipfs/go-metrics-interface"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/fxex"
)

var (
	_ common.StateManager = (*stmgr.StateManager)(nil)
	_ common.ChainStore   = (*store.ChainStore)(nil)
)

// DefaultBellProvider combines the providers for basic components inside bell
var DefaultBellProvider = fx.Provide(
	func() vm.SyscallBuilder {
		return vm.Syscalls(ffiwrapper.ProofVerifier)
	},

	// from lotus
	journal.NilJournal,
	modules.ChainStore,
	stmgr.DefaultUpgradeSchedule,
	stmgr.NewStateManagerWithUpgradeSchedule,

	// basics
	MgoMetaMgr,
	MgoHeadNotifier,
	NewRaCailum,

	// type convertion
	fxex.Convert(new(dtypes.HotBlockstore), new(dtypes.ChainBlockstore)),
	fxex.Convert(new(dtypes.HotBlockstore), new(dtypes.StateBlockstore)),
	fxex.Convert(new(dtypes.HotBlockstore), new(dtypes.BaseBlockstore)),
	fxex.Convert(new(dtypes.HotBlockstore), new(dtypes.ExposedBlockstore)),
	fxex.Convert(new(*store.ChainStore), new(common.ChainStore)),
	fxex.Convert(new(*stmgr.StateManager), new(common.StateManager)),
)

// BellApp constructs a *fx.App with givent opts and defaults
func BellApp(ctx context.Context, logger fx.Printer, target interface{}, opts ...fx.Option) *fx.App {
	opts = append([]fx.Option{
		// raw stores should be readonly
		fxex.ProvideEx(
			fxex.As(metricsi.CtxScope(ctx, "bell"), new(helpers.MetricsCtx)),
			fxex.As(ctx, new(GlobalContext)),
		),
		DefaultBellProvider,
	}, opts...)

	if logger != nil {
		opts = append(opts, fx.Logger(logger))
	}

	if target != nil {
		opts = append(opts, fx.Populate(target))
	}

	return fx.New(opts...)
}
