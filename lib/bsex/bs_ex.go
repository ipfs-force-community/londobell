package bsex

import (
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
)

var _ blockstore.Blockstore = (*CachedBlockstore)(nil)

func NewCachedBlockstore(cacheSize int, bs blockstore.Blockstore) (*CachedBlockstore, error) {
	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	return &CachedBlockstore{
		cache:      cache,
		Blockstore: bs,
	}, nil
}

type CachedBlockstore struct {
	cache *lru.TwoQueueCache
	blockstore.Blockstore
}

func (cbs *CachedBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return b, nil
		}
	}

	b, err := cbs.Blockstore.Get(c)
	if err != nil {
		return nil, err
	}

	cbs.cache.Add(c, b)
	return b, nil
}

func (cbs *CachedBlockstore) Has(c cid.Cid) (bool, error) {
	if has := cbs.cache.Contains(c); has {
		return true, nil
	}

	return cbs.Blockstore.Has(c)
}
