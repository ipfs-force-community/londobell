package bsex

import (
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go4.org/syncutil/singleflight"

	"github.com/filecoin-project/lotus/blockstore"

	"github.com/dtynn/londobell/prometheus"
)

var log = logging.Logger("bs_ex")

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
	prometheus.CacheGetCnt.Inc()
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return b, nil
		}
	}

	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		prometheus.CacheGetMissCnt.Inc()
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
	prometheus.CacheViewCnt.Inc()
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return callback(b.RawData())
		}
	}
	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		prometheus.CacheViewMissCnt.Inc()
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
	prometheus.CacheHasCnt.Inc()
	if has := cbs.cache.Contains(c); has {
		return true, nil
	}
	b, err := cbs.hasSg.Do(c.String(), func() (interface{}, error) {
		prometheus.CacheHasMissCnt.Inc()
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
