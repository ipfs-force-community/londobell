package bsex

import (
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go4.org/syncutil/singleflight"

	"github.com/filecoin-project/lotus/blockstore"
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
	go res.Stat()
	return res, nil
}

func (cbs *CachedBlockstore) Stat() {
	timer := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-timer.C:
		}

		log.Infof("CachedBlockstore miss stat: Get: %d %d, View %d %d, Has %d %d\n", cbs.getMiss, cbs.getCnt,
			cbs.viewMiss, cbs.viewCnt, cbs.hasMiss, cbs.hasCnt)
	}
}

type CachedBlockstore struct {
	cache *lru.TwoQueueCache
	blockstore.Blockstore

	getSg *singleflight.Group
	hasSg *singleflight.Group

	getMiss, getCnt, viewMiss, viewCnt, hasMiss, hasCnt int64
}

func (cbs *CachedBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	atomic.AddInt64(&cbs.getCnt, 1)
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return b, nil
		}
	}

	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		atomic.AddInt64(&cbs.getMiss, 1)
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
	atomic.AddInt64(&cbs.viewCnt, 1)
	if cached, has := cbs.cache.Get(c); has {
		if b, ok := cached.(blocks.Block); ok {
			return callback(b.RawData())
		}
	}
	b, err := cbs.getSg.Do(c.String(), func() (interface{}, error) {
		atomic.AddInt64(&cbs.viewMiss, 1)
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
	atomic.AddInt64(&cbs.hasCnt, 1)
	if has := cbs.cache.Contains(c); has {
		return true, nil
	}
	b, err := cbs.hasSg.Do(c.String(), func() (interface{}, error) {
		atomic.AddInt64(&cbs.hasMiss, 1)
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
