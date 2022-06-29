package dep

import (
	"context"
	"net/http"

	"github.com/dtynn/dix"
	metricsi "github.com/ipfs/go-metrics-interface"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/beacon"

	"github.com/filecoin-project/lotus/build"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/cliex"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/racailum/segment"
)

var (
	_ common.StateManager = (*stmgr.StateManager)(nil)
	_ common.ChainStore   = (*store.ChainStore)(nil)
)

const (
	invokeNone dix.Invoke = iota // nolint: varcheck,deadcode

	invokeSetupGrafana
	invokeSetupDebug
	invokeSetupMetrics
	invokeSetupTracing

	invokePopulate
)

func Bell(ctx context.Context, logger fx.Printer, target ...interface{}) dix.Option {
	return dix.Options(
		dix.Override(new(GlobalContext), ctx),
		dix.Override(new(*http.ServeMux), http.NewServeMux()),
		dix.Override(new(helpers.MetricsCtx), metricsi.CtxScope(ctx, "bell")),

		dix.If(logger != nil, dix.Logger(logger)),
		dix.If(len(target) > 0, dix.Populate(invokePopulate, target...)),

		dix.Override(new(racailum.Config), LoadRaConfig),
		dix.Override(new(SegmentMetaDS), OpenSegmentDS),
		dix.Override(new(*segment.Manager), NewSegmentManager),

		dix.Override(new(vm.SyscallBuilder), vm.Syscalls(ffiwrapper.ProofVerifier)),
		dix.Override(new(journal.Journal), journal.NilJournal),
		dix.Override(new(store.WeightFunc), filcns.Weight),
		dix.Override(new(*store.ChainStore), modules.ChainStore),
		dix.Override(new(stmgr.Executor), filcns.NewTipSetExecutor),
		dix.Override(new(dtypes.DrandSchedule), modules.BuiltinDrandConfig),
		dix.Override(new(dtypes.AfterGenesisSet), NewAfterGenesisSet),
		dix.Override(new(beacon.Schedule), modules.RandomSchedule),
		dix.Override(new(stmgr.UpgradeSchedule), filcns.DefaultUpgradeSchedule),
		dix.Override(new(*stmgr.StateManager), stmgr.NewStateManager),
		dix.Override(new(modules.Genesis), modules.LoadGenesis(build.MaybeGenesis())),
		dix.Override(new(common.HeadNotifier), cliex.NewHeadSub),
		dix.Override(new(*racailum.RaCailum), NewRaCailum),
		dix.Override(new(repo.Repo), repo.NewMemory(nil)),
		dix.Override(new(repo.LockedRepo), LockedRepo),
		dix.Override(new(dtypes.MetadataDS), InMemMetadataDS),

		dix.Override(new(dtypes.HotBlockstore), ChainIOBlockstore),
		dix.Override(new(dtypes.ChainBlockstore), dix.From(new(dtypes.HotBlockstore))),
		dix.Override(new(dtypes.StateBlockstore), dix.From(new(dtypes.HotBlockstore))),
		dix.Override(new(dtypes.BaseBlockstore), dix.From(new(dtypes.HotBlockstore))),
		dix.Override(new(dtypes.ExposedBlockstore), dix.From(new(dtypes.HotBlockstore))),

		dix.Override(new(common.ChainStore), dix.From(new(*store.ChainStore))),
		dix.Override(new(common.StateManager), dix.From(new(*stmgr.StateManager))),

		dix.Override(invokeSetupDebug, SetupPprof),
		dix.Override(invokeSetupMetrics, SetupMetric),
		dix.Override(invokeSetupTracing, SetupTracing),
		dix.Override(invokeSetupGrafana, SetupGrafana),
	)
}
