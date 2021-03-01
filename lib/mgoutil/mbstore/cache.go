package mbstore

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"golang.org/x/xerrors"
)

var (
	_ ChainIOEx = (*ShardedLRUChainIO)(nil)
	_ ChainIOEx = (*LRUChainIO)(nil)
	_ ChainIOEx = (*RedisChainIO)(nil)
)

// NewShardedLRUChainIO returns a ChainIOEx with given shard num and entry size
func NewShardedLRUChainIO(shard, size int) (ChainIOEx, error) {
	if shard == 0 {
		return nil, xerrors.Errorf("shard should be greater than zero")
	}

	if shard == 1 {
		return NewLRUChainIO(size)
	}

	all := make([]*LRUChainIO, 0, shard)
	for i := 0; i < shard; i++ {
		l, err := NewLRUChainIO(size)
		if err != nil {
			return nil, xerrors.Errorf("init #%d lru: %w", i, err)
		}

		all = append(all, l)
	}

	return &ShardedLRUChainIO{shard: all}, nil
}

// ShardedLRUChainIO use a group of LRUChainIO to split objects
type ShardedLRUChainIO struct {
	shard []*LRUChainIO
}

func (s *ShardedLRUChainIO) choice(c cid.Cid) *LRUChainIO {
	return s.shard[int(crc32.ChecksumIEEE(c.Bytes()))%len(s.shard)]
}

// ChainReadObj impls ChainIOEx
func (s *ShardedLRUChainIO) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	return s.choice(c).ChainReadObj(ctx, c)
}

// ChainHasObj impls ChainIOEx
func (s *ShardedLRUChainIO) ChainHasObj(ctx context.Context, c cid.Cid) (bool, error) {
	return s.choice(c).ChainHasObj(ctx, c)
}

// ChainPutObj impls ChainIOEx
func (s *ShardedLRUChainIO) ChainPutObj(ctx context.Context, c cid.Cid, data []byte) error {
	return s.choice(c).ChainPutObj(ctx, c, data)
}

// ChainRemoveObj impls ChainIOEx
func (s *ShardedLRUChainIO) ChainRemoveObj(ctx context.Context, c cid.Cid) error {
	return s.choice(c).ChainRemoveObj(ctx, c)
}

// NewLRUChainIO returns a LRUChainIO with given entry size
func NewLRUChainIO(size int) (*LRUChainIO, error) {
	cache, err := lru.New2Q(size)
	if err != nil {
		return nil, err
	}

	return &LRUChainIO{
		cache: cache,
	}, nil
}

// LRUChainIO is an implementation of ChainIOEx based on lru cache
type LRUChainIO struct {
	cache *lru.TwoQueueCache
}

// ChainReadObj impls ChainIOEx
func (lc *LRUChainIO) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	data, ok := lc.cache.Get(c)
	if !ok {
		return nil, blockstore.ErrNotFound
	}

	return data.([]byte), nil
}

// ChainHasObj impls ChainIOEx
func (lc *LRUChainIO) ChainHasObj(ctx context.Context, c cid.Cid) (bool, error) {
	return lc.cache.Contains(c), nil
}

// ChainPutObj impls ChainIOEx
func (lc *LRUChainIO) ChainPutObj(ctx context.Context, c cid.Cid, data []byte) error {
	lc.cache.Add(c, data)
	return nil
}

// ChainRemoveObj impls ChainIOEx
func (lc *LRUChainIO) ChainRemoveObj(ctx context.Context, c cid.Cid) error {
	lc.cache.Remove(c)
	return nil
}

// NewRedisChainIO returns a LRUChainIO with given data source name
func NewRedisChainIO(dsn string) (*RedisChainIO, error) {
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, err
	}

	return &RedisChainIO{
		cli: redis.NewClient(opt),
	}, nil
}

// RedisChainIO is an implementation of ChainIOEx based on redis
type RedisChainIO struct {
	cli *redis.Client
}

// ChainReadObj impls ChainIOEx
func (rc *RedisChainIO) ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error) {
	data, err := rc.cli.Get(ctx, c.String()).Bytes()
	if err == nil {
		return data, nil
	}

	if err == redis.Nil {
		return nil, blockstore.ErrNotFound
	}

	return nil, err
}

// ChainHasObj impls ChainIOEx
func (rc *RedisChainIO) ChainHasObj(ctx context.Context, c cid.Cid) (bool, error) {
	res, err := rc.cli.Exists(ctx, c.String()).Result()
	return res > 0, err
}

// ChainPutObj impls ChainIOEx
func (rc *RedisChainIO) ChainPutObj(ctx context.Context, c cid.Cid, data []byte) error {
	if err := rc.cli.Set(ctx, c.String(), data, 30*time.Minute).Err(); err != nil {
		return err
	}

	return nil
}

// ChainRemoveObj impls ChainIOEx
func (rc *RedisChainIO) ChainRemoveObj(ctx context.Context, c cid.Cid) error {
	return rc.cli.Del(ctx, c.String()).Err()
}
