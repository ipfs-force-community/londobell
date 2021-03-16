package dep

import (
	"context"

	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil"
	"github.com/dtynn/londobell/lib/mgoutil/mbstore"
	"github.com/dtynn/londobell/lib/mgoutil/mmetads"
	"github.com/dtynn/londobell/lib/mgoutil/mmetads/headsub"
	"github.com/dtynn/londobell/lib/mgoutil/mmetamgr"
	"github.com/dtynn/londobell/racailum"
	"github.com/dtynn/londobell/racailum/segment"
)

type metadsIn struct {
	fx.In
	DSN MgoMetaDSDSN
	RO  MgoMetaDSReadOnly `optional:"true"`
}

// NewMgoMetaDSClient attempts to establish a mongo client for metads
func NewMgoMetaDSClient(ctx GlobalContext, in metadsIn) (MgoMetaDSClient, error) {
	cli, err := mgoutil.Connect(ctx, string(in.DSN))
	return cli, err
}

// MgoMetaDS provides a MgoDS as the dtypes.MetadataDS
func MgoMetaDS(mctx helpers.MetricsCtx, lc fx.Lifecycle, cli MgoMetaDSClient, in metadsIn) (dtypes.MetadataDS, error) {
	ctx := helpers.LifecycleCtx(mctx, lc)

	log.Infow("constructing mgo MetadataDS", "read-only", in.RO)
	mds, err := mmetads.NewMgoDS(ctx, cli, bool(in.RO))
	return mds, err
}

type chainRawBlockstoreIn struct {
	fx.In
	DSN  MgoBstoreDSN
	Sync MgoBstoreSync     `optional:"true"`
	RO   MgoBstoreReadOnly `optional:"true"`
}

// MgoChainHotBlockstore provides a MgoBstore as the dtypes.HotBlockstore
func MgoChainHotBlockstore(mctx helpers.MetricsCtx, lc fx.Lifecycle, in chainRawBlockstoreIn) (dtypes.HotBlockstore, error) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	mb, err := mbstore.NewMgoBstore(ctx, string(in.DSN), mbstore.MgoPutSync(bool(in.Sync)), mbstore.MgoReadOnly(bool(in.RO)))
	if err != nil {
		return nil, err

	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			mb.Run()
			return nil

		},
		OnStop: func(context.Context) error {
			mb.Stop()
			return nil

		},
	})

	var mopts mbstore.MixedStoreOptions

	lcache, err := mbstore.NewShardedLRUChainIO(1, 4<<20)
	if err != nil {
		return nil, err

	}

	bs, err := mbstore.NewMixedBstore(ctx, mb, lcache, mopts)
	if err != nil {
		return nil, err

	}

	log.Infow("constructing mgo ChainRawBlockstore", "sync", in.Sync, "read-only", in.RO)
	return blockstore.WrapIDStore(bs), nil
}

// MgoMetaMgr provides a MetaMgr as the common.MetaManager
func MgoMetaMgr(ctx GlobalContext, dsn MgoMetaMgrDSN) (common.MetaManager, error) {
	mgr, err := mmetamgr.New(ctx, string(dsn))
	return mgr, err
}

// MgoHeadNotifier provides a common.HeadNotifier based on metads
func MgoHeadNotifier(cli MgoMetaDSClient) (common.HeadNotifier, error) {
	sub, err := headsub.New(cli)
	return sub, err
}

type raIn struct {
	fx.In
	Ctx     GlobalContext
	Cfg     racailum.Config
	Sub     common.HeadNotifier
	MetaMgr common.MetaManager
	CS      common.ChainStore
	Stm     common.StateManager
	SegOpts []segment.OptionFn `group:"segopt"`
}

// NewRaCailum constructs an instance of RaCailum
func NewRaCailum(in raIn) (*racailum.RaCailum, error) {
	return racailum.New(in.Ctx, in.Cfg, in.Sub, in.MetaMgr, in.CS, in.Stm, in.SegOpts...)
}

// SegmentOpt is used to combine a group of option funcs
type SegmentOpt struct {
	fx.Out
	Opt segment.OptionFn `group:"segopt"`
}
