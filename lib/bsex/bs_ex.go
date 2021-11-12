package bsex

import (
	"context"

	blocks "github.com/ipfs/go-block-format"

	"github.com/filecoin-project/lotus/blockstore"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs/go-cid"
	"go4.org/syncutil/singleflight"
)

var _ blockstore.Blockstore = (*CachedBlockstore)(nil)

func NewCachedBlockstore(cacheSize int, bs blockstore.Blockstore) (*CachedBlockstore, error) {
	cache, err := lru.New2Q(cacheSize)
	if err != nil {
		return nil, err
	}

	getSg := new(singleflight.Group)
	hasSg := new(singleflight.Group)
	res := &CachedBlockstore{
		cache:      cache,
		Blockstore: bs,

		getSg: getSg,
		hasSg: hasSg,
	}

	return res, nil
}

type CachedBlockstore struct {
	cache *lru.TwoQueueCache
	blockstore.Blockstore

	getSg *singleflight.Group
	hasSg *singleflight.Group
}

func (cbs *CachedBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	metrics.RecordInc(context.Background(), metrics.CacheGetCnt)
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return b, nil
		}
	}

	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		metrics.RecordInc(context.Background(), metrics.CacheGetMissCnt)
		b, err := cbs.Blockstore.Get(c)
		if err != nil {
			return nil, err
		}

		cbs.cache.Add(c, b)
		return b, nil
	})

	if err != nil {
		return nil, err
	}

	return b.(blocks.Block), nil
}

func (cbs *CachedBlockstore) View(c cid.Cid, callback func([]byte) error) error {
	metrics.RecordInc(context.Background(), metrics.CacheViewCnt)

	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return callback(b.RawData())
		}
	}
	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		metrics.RecordInc(context.Background(), metrics.CacheViewMissCnt)
		b, err := cbs.Blockstore.Get(c)
		if err != nil {
			return nil, err
		}

		cbs.cache.Add(c, b)
		return b, nil
	})

	if err != nil {
		return err
	}

	return callback(b.(blocks.Block).RawData())
}

func (cbs *CachedBlockstore) Has(c cid.Cid) (bool, error) {
	metrics.RecordInc(context.Background(), metrics.CacheHasCnt)

	if has := cbs.cache.Contains(c); has {
		return true, nil
	}
	b, err := cbs.hasSg.Do(c.String(), func() (interface{}, error) {
		metrics.RecordInc(context.Background(), metrics.CacheHasMissCnt)
		b, err := cbs.Blockstore.Has(c)
		if err != nil {
			return false, err
		}
		return b, nil
	})

	if err != nil {
		return false, err
	}

	return b.(bool), nil
}
