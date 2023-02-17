package testutils

import (
	"context"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/filecoin-project/lotus/api/client"
	badgerbs "github.com/filecoin-project/lotus/blockstore/badger"
	"github.com/filecoin-project/lotus/node/repo"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-libipfs/blocks"
	"github.com/ipfs/go-merkledag"

	bstore "github.com/filecoin-project/lotus/blockstore"
)

func NewApiBlockStore(ctx context.Context, url string) (bstore.Blockstore, error) {
	fullNode, _, err := client.NewFullNodeRPCV0(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	bs := bstore.NewAPIBlockstore(fullNode)
	return bs, err
}

func NewLocalBlockStore(ctx context.Context) (bstore.Blockstore, func() error, error) {
	p := getTestDBPath()
	opts, _ := repo.BadgerBlockstoreOptions("test", path.Join(p), false)
	if nosyncBs, nosyncBsSet := os.LookupEnv("LOTUS_CHAIN_BADGERSTORE_DISABLE_FSYNC"); nosyncBsSet {
		nosyncBs = strings.ToLower(nosyncBs)
		if nosyncBs == "" || nosyncBs == "0" || nosyncBs == "false" || nosyncBs == "no" {
			opts.SyncWrites = true
		} else {
			opts.SyncWrites = false
		}
	}

	bs, err := badgerbs.Open(opts)
	if err != nil {
		return nil, nil, err
	}
	bbs := bstore.WrapIDStore(bs)
	return bbs, bs.Close, nil
}

func GenerateFullTree(ctx context.Context, root cid.Cid, sourceBs bstore.Blockstore, localBs bstore.Blockstore) error {
	bsvc := blockservice.New(sourceBs, offline.Exchange(sourceBs))
	dag := merkledag.NewDAGService(bsvc)
	seen := cid.NewSet()
	res := make([]blocks.Block, 0)
	var statslk sync.Mutex

	// walker: visit and save all child node from root node
	walker := func(ctx context.Context, c cid.Cid) ([]*ipld.Link, error) {
		if c.Prefix().Codec == cid.FilCommitmentSealed || c.Prefix().Codec == cid.FilCommitmentUnsealed {
			return []*ipld.Link{}, nil
		}

		nd, err := dag.Get(ctx, c)
		if err != nil {
			return nil, err
		}

		blk, err := blocks.NewBlockWithCid(nd.RawData(), c)
		if err != nil {
			return []*ipld.Link{}, err
		}
		statslk.Lock()
		res = append(res, blk)
		statslk.Unlock()
		return nd.Links(), nil
	}

	err := merkledag.Walk(ctx, walker, root, seen.Visit, merkledag.Concurrent())
	if err != nil {
		return err
	}
	err = localBs.PutMany(ctx, res)
	if err != nil {
		return err
	}
	return nil
}

// getTestDBPath Only called by NewLocalBlockStore.
// It may not get correct path if called by other callers.
func getTestDBPath() string {
	_, filename, _, _ := runtime.Caller(1)
	p := path.Dir(path.Dir(filename))
	return path.Join(p, "testdb")
}
