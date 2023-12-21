package dep

import (
	"context"
	"net/http"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/beacon"
	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/index"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/storage/sealer/ffiwrapper"
	"github.com/filecoin-project/lotus/storage/sealer/storiface"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/cliex"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs/go-datastore"
	metricsi "github.com/ipfs/go-metrics-interface"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/tmpbell"
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

func ContextModule(ctx context.Context) dix.Option {
	return dix.Options(dix.Override(new(GlobalContext), ctx),
		dix.Override(new(*http.ServeMux), http.NewServeMux()),
		dix.Override(new(helpers.MetricsCtx), metricsi.CtxScope(ctx, "bell")))
}

func SegmentManager() dix.Option {
	return dix.Options(dix.Override(new(racailum.Config), LoadRaConfig),
		dix.Override(new(SegmentMetaDS), OpenSegmentDS),
		dix.Override(new(*segment.Manager), NewSegmentManager))
}

func StateManager() dix.Option {
	return dix.Options(dix.Override(new(vm.SyscallBuilder), vm.Syscalls),
		dix.Override(new(storiface.Verifier), ffiwrapper.ProofVerifier),
		dix.Override(new(journal.Journal), modules.OpenFilesystemJournal),
		dix.Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		dix.Override(new(store.WeightFunc), filcns.Weight),
		dix.Override(new(*store.ChainStore), modules.ChainStore),
		dix.Override(new(stmgr.Executor), consensus.NewTipSetExecutor(filcns.RewardFunc)),
		dix.Override(new(dtypes.DrandSchedule), BuiltinDrandConfig),
		dix.Override(new(dtypes.AfterGenesisSet), modules.SetGenesis),
		dix.Override(new(beacon.Schedule), modules.RandomSchedule),
		dix.Override(new(stmgr.UpgradeSchedule), modules.UpgradeSchedule),
		dix.Override(new(datastore.Batching), InMemMetadataDS),
		// 需配合节点设置MsgIndex或DummyMsgIndex
		// dix.Override(new(index.MsgIndex), modules.MsgIndex),
		dix.Override(new(index.MsgIndex), modules.DummyMsgIndex),
		dix.Override(new(*stmgr.StateManager), stmgr.NewStateManager),
		dix.Override(new(modules.Genesis), modules.LoadGenesis(build.MaybeGenesis())),
		dix.Override(new(dtypes.ChainBlockstore), dix.From(new(dtypes.BasicChainBlockstore))),
		dix.Override(new(dtypes.StateBlockstore), dix.From(new(dtypes.BasicChainBlockstore))),
		dix.Override(new(dtypes.BaseBlockstore), dix.From(new(dtypes.BasicChainBlockstore))),
		dix.Override(new(dtypes.ExposedBlockstore), dix.From(new(dtypes.BasicChainBlockstore))),
		dix.Override(new(common.ChainStore), dix.From(new(*store.ChainStore))),
		dix.Override(new(common.StateManager), dix.From(new(*stmgr.StateManager))))
}

func OnlineDataSource() dix.Option {
	return dix.Options(
		dix.Override(new(dtypes.BasicChainBlockstore), ChainIOBlockstore),
		dix.Override(new(repo.Repo), repo.NewMemory(nil)),
		dix.Override(new(repo.LockedRepo), LockedRepo),
		dix.Override(new(dtypes.MetadataDS), InMemMetadataDS))
}

func OfflineDataSource() dix.Option {
	return dix.Options(
		// Notice: we may need to use other datastore someday. It depends on
		// the origin data structs.
		dix.Override(new(dtypes.AfterGenesisSet), modules.SetGenesis),
		dix.Override(new(dtypes.BasicChainBlockstore), ChainOfflineBlockstore),
		dix.Override(new(dtypes.MetadataDS), modules.Datastore(true)),
	)
}

func Bell(ctx context.Context, logger fx.Printer, target ...interface{}) dix.Option {
	return dix.Options(
		ContextModule(ctx),

		dix.If(logger != nil, dix.Logger(logger)),
		dix.If(len(target) > 0, dix.Populate(invokePopulate, target...)),

		SegmentManager(),

		StateManager(),

		OnlineDataSource(),

		// londo bell module
		dix.Override(new(*racailum.RaCailum), NewRaCailum),
		dix.Override(new(*tmpbell.TmpBell), NewTmpBell), //tmp db
		dix.Override(new(common.HeadNotifier), cliex.NewHeadSub),

		dix.Override(invokeSetupDebug, SetupPprof),
		dix.Override(invokeSetupMetrics, SetupMetric),
		dix.Override(invokeSetupTracing, SetupTracing),
		dix.Override(invokeSetupGrafana, SetupGrafana),
	)
}

func WalkRaCalium(cctx *cli.Context, logger fx.Printer, target ...interface{}) dix.Option {
	return dix.Options(
		ContextModule(context.Background()),

		dix.If(logger != nil, dix.Logger(logger)),
		dix.If(len(target) > 0, dix.Populate(invokePopulate, target...)),
		SegmentManager(),
		StateManager(),

		dix.ApplyIf(func(s *dix.Settings) bool {
			return cctx.Bool("local")
		}, InjectChainRepo(cctx), OfflineDataSource()),

		dix.ApplyIf(func(s *dix.Settings) bool {
			return !cctx.Bool("local")
		}, InjectFullNode(cctx), OnlineDataSource()),

		dix.Override(new(*racailum.RaCailum), NewRaCailum),
		dix.Override(new(common.HeadNotifier), func() (common.HeadNotifier, error) { return nil, nil }),
	)
}
