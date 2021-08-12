package dep

import (
	"context"
	"os"
	"strconv"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	dstore "github.com/ipfs/go-datastore"
	"go.uber.org/fx"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/bsex"
	"github.com/dtynn/londobell/lib/cliex"
	"github.com/dtynn/londobell/lib/mgoutil"
	"github.com/dtynn/londobell/lib/mgoutil/mmetamgr"
	"github.com/dtynn/londobell/racailum"
	"github.com/dtynn/londobell/racailum/segment"
)

var (
	_ common.MetaManager  = (*mmetamgr.MetaMgr)(nil)
	_ common.DocumentDB   = (*mgoutil.MgoDocDB)(nil)
	_ common.HeadNotifier = (*cliex.HeadSub)(nil)
)

// HeadNotifier provides a common.HeadNotifier based on metads
func HeadNotifier(cli v0api.FullNode) (common.HeadNotifier, error) {
	sub, err := cliex.NewHeadSub(cli)
	return sub, err
}

type WrapAPIBlockstore struct {
	blockstore.Blockstore
}

func (a *WrapAPIBlockstore) Put(blocks.Block) error {
	return nil
}

func (a *WrapAPIBlockstore) PutMany([]blocks.Block) error {
	return nil
}

func (a *WrapAPIBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, nil
}

func (a *WrapAPIBlockstore) DeleteBlock(cid.Cid) error {
	return nil
}

func ChainIOBlockstore(full v0api.FullNode) (dtypes.HotBlockstore, error) {
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

type raIn struct {
	fx.In
	Ctx    GlobalContext
	Cfg    racailum.Config
	Sub    common.HeadNotifier
	CS     common.ChainStore
	Stm    common.StateManager
	SegMgr *segment.Manager
}

// NewRaCailum constructs an instance of RaCailum
func NewRaCailum(in raIn) (*racailum.RaCailum, error) {
	return racailum.New(in.Ctx, in.Cfg, in.Sub, in.CS, in.Stm, in.SegMgr)
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
	err = ds.Put(dstore.NewKey("0"), bh.Cid().Bytes())
	return ds, err
}
