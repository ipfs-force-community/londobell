package mbstore

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"

	"github.com/filecoin-project/lotus/api"
)

// ChainIO is a type alias to the apibstore.ChainIO
type ChainIO = api.ChainIO

// ChainIOEx extends the ChainIO with some write methods
type ChainIOEx interface {
	ChainIO
	ChainPutObj(context.Context, cid.Cid, []byte) error
	ChainRemoveObj(context.Context, cid.Cid) error
}

var _ blockstore.Blockstore = (*MixedBstore)(nil)

// MixedStoreOptions is the options
type MixedStoreOptions struct {
}

// NewMixedBstore mixes multiple levels to construct a blockstore.Blockstore
func NewMixedBstore(ctx context.Context, lower blockstore.Blockstore, cache ChainIOEx, opts MixedStoreOptions) (*MixedBstore, error) {
	return &MixedBstore{
		cache: cache,
		lower: lower,
		opts:  opts,
		ctx:   ctx,
	}, nil
}

// MixedBstore is an implementation of blockstore.Blockstore based on multiple levels
type MixedBstore struct {
	cache ChainIOEx // multiple levels of cache instance, like local lru, redis, etc
	lower blockstore.Blockstore
	opts  MixedStoreOptions

	ctx context.Context
}

// DeleteBlock impls blockstore.Blockstore
func (mb *MixedBstore) DeleteBlock(c cid.Cid) error {
	err := mb.lower.DeleteBlock(c)
	if err != nil {
		return err
	}

	mb.cache.ChainRemoveObj(mb.ctx, c)
	return nil
}

// Has impls blockstore.Blockstore
func (mb *MixedBstore) Has(c cid.Cid) (bool, error) {
	_, err := mb.Get(c)
	if err == nil {
		return true, nil
	}

	if err == blockstore.ErrNotFound {
		return false, nil
	}

	return false, err
}

// Get impls blockstore.Blockstore
func (mb *MixedBstore) Get(c cid.Cid) (blocks.Block, error) {
	if data, _ := mb.cache.ChainReadObj(mb.ctx, c); len(data) > 0 {
		b, err := blocks.NewBlockWithCid(data, c)
		if err == nil {
			return b, nil
		}
	}

	b, err := mb.lower.Get(c)
	if err != nil {
		return nil, err
	}

	mb.cache.ChainPutObj(mb.ctx, c, b.RawData())

	return b, nil
}

// GetSize impls blockstore.Blockstore
func (mb *MixedBstore) GetSize(c cid.Cid) (int, error) {
	b, err := mb.Get(c)
	if err != nil {
		return 0, err
	}

	return len(b.RawData()), nil
}

// Put impls blockstore.Blockstore
func (mb *MixedBstore) Put(b blocks.Block) error {
	return mb.PutMany([]blocks.Block{b})
}

// PutMany impls blockstore.Blockstore
func (mb *MixedBstore) PutMany(bs []blocks.Block) error {
	if err := mb.lower.PutMany(bs); err != nil {
		return err
	}

	for bi := range bs {
		id := bs[bi].Cid()
		mb.cache.ChainPutObj(mb.ctx, id, bs[bi].RawData())
	}

	return nil
}

// AllKeysChan impls blockstore.Blockstore
func (mb *MixedBstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return mb.lower.AllKeysChan(ctx)
}

// HashOnRead impls blockstore.Blockstore
func (mb *MixedBstore) HashOnRead(enabled bool) {
	mb.lower.HashOnRead(enabled)
}
