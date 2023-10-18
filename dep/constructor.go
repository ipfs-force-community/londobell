package dep

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/filecoin-project/lotus/node/modules/helpers"

	bf "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	dstore "github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node/modules"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/bsex"
	"github.com/ipfs-force-community/londobell/lib/cliex"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mmetamgr"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/racailum/debug"
	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs-force-community/londobell/racailum/tracing"
	"github.com/ipfs-force-community/londobell/tmpbell"
)

var (
	_ common.MetaManager  = (*mmetamgr.MetaMgr)(nil)
	_ common.DocumentDB   = (*mgoutil.MgoDocDB)(nil)
	_ common.HeadNotifier = (*cliex.HeadSub)(nil)
)

type WrapAPIBlockstore struct {
	blockstore.Blockstore
}

func (a *WrapAPIBlockstore) Put(context.Context, bf.Block) error {
	return nil
}

func (a *WrapAPIBlockstore) PutMany(context.Context, []bf.Block) error {
	return nil
}

func (a *WrapAPIBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, nil
}

func (a *WrapAPIBlockstore) DeleteBlock(context.Context, cid.Cid) error {
	return nil
}

func ChainIOBlockstore(full v0api.FullNode) (dtypes.BasicChainBlockstore, error) {
	bs := blockstore.NewAPIBlockstore(full)
	wrapBlockStore := &WrapAPIBlockstore{
		bs,
	}

	cacheSize := 1 << 25
	if size := os.Getenv("BELL_CACHE_SIZE"); size != "" {
		var err error
		cacheSize, err = strconv.Atoi(size)
		if err != nil {
			panic(err)
		}
	}

	cached, err := bsex.NewCachedBlockstore(cacheSize, wrapBlockStore)
	if err != nil {
		return nil, err
	}

	return cached, nil
}

func ChainOfflineBlockstore(lc fx.Lifecycle, mctx helpers.MetricsCtx, r repo.LockedRepo, writableOffline WritableOffline) (dtypes.BasicChainBlockstore, error) {
	bs, err := r.Blockstore(helpers.LifecycleCtx(mctx, lc), repo.UniversalBlockstore)
	if err != nil {
		return nil, err
	}

	var wrapBlockStore dtypes.BasicChainBlockstore
	if writableOffline {
		wrapBlockStore = bs
	} else {
		wrapBlockStore = &WrapAPIBlockstore{
			bs,
		}
	}

	if c, ok := bs.(io.Closer); ok {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				return c.Close()
			},
		})
	}
	return wrapBlockStore, err
}

type raIn struct {
	fx.In
	Ctx        GlobalContext
	Cfg        racailum.Config
	Sub        common.HeadNotifier
	CS         common.ChainStore
	Stm        common.StateManager
	SegMgr     *segment.Manager
	ShutDownCh dtypes.ShutdownChan
}

// NewRaCailum constructs an instance of RaCailum
func NewRaCailum(in raIn) (*racailum.RaCailum, error) {
	return racailum.New(in.Ctx, in.Cfg, in.Sub, in.CS, in.Stm, in.SegMgr, in.ShutDownCh)
}

type tmpIn struct {
	fx.In
	Ctx    GlobalContext
	Cfg    racailum.Config
	CS     common.ChainStore
	Stm    common.StateManager
	SegMgr *segment.Manager
	Full   v0api.FullNode
}

func NewTmpBell(in tmpIn) (*tmpbell.TmpBell, error) {
	return tmpbell.New(in.Ctx, in.Cfg, in.CS, in.Stm, in.SegMgr, in.Full)
}

// SegmentOpt is used to combine a group of option funcs
type SegmentOpt struct {
	fx.Out
	Opt segment.OptionFn `group:"segopt"`
}

func InMemRepo() repo.Repo {
	return repo.NewMemory(nil)
}

func LockedRepo(r repo.Repo) (repo.LockedRepo, error) {
	return r.Lock(repo.FullNode)
}

func InMemMetadataDS(lr repo.LockedRepo, g modules.Genesis) (dtypes.MetadataDS, error) {
	ds, err := lr.Datastore(context.Background(), "inmem")
	if err != nil {
		return nil, err
	}
	bh, err := g()
	if err != nil {
		return nil, err
	}
	err = ds.Put(context.Background(), dstore.NewKey("0"), bh.Cid().Bytes())
	return ds, err
}

func LoadRaConfig(rpath RepoPath) (racailum.Config, error) {
	cfgPath := ConfigFilePath(rpath)
	cfg := racailum.DefaultConfig()
	opt := config.SetDefault(func() (interface{}, error) {
		return cfg, nil
	})
	_, err := config.FromFile(cfgPath, opt)
	if err != nil {
		return racailum.Config{}, fmt.Errorf("read config from file %s: %w", cfgPath, err)
	}

	return cfg, nil
}

func OpenSegmentDS(rpath RepoPath) (SegmentMetaDS, error) {
	return levelDs(SegmentMetaDSPath(rpath), false)
}

func levelDs(path string, readonly bool) (dtypes.MetadataDS, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func NewSegmentManager(segds SegmentMetaDS) (*segment.Manager, error) {
	return segment.NewManager(segds)
}

func SetupPprof(cfg racailum.Config, mux *http.ServeMux) error {
	if cfg.EnableDebug {
		debug.Setup(mux)
	}

	return nil
}

func SetupMetric(cfg racailum.Config, mux *http.ServeMux) error {
	return metrics.Setup(&cfg.Metrics, mux)
}

func SetupTracing(lc fx.Lifecycle, cfg racailum.Config, mux *http.ServeMux) error {
	je := tracing.Setup(&cfg.Tracing, mux)
	if je == nil {
		return nil
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			je.ForceFlush(context.Background())
			return nil
		},
	})

	return nil
}

func SetupGrafana(cfg racailum.Config, mux *http.ServeMux) error {
	// TODO: move grafana setup here
	return nil
}
