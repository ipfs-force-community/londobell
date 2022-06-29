package testutils

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/blockstore"
)

type MockChainIO struct {
	blockstore.Blockstore
}

func (cio *MockChainIO) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	blk, err := cio.Get(ctx, obj)
	if err != nil {
		return []byte{}, err
	}
	return blk.RawData(), nil
}

func (cio *MockChainIO) ChainHasObj(ctx context.Context, obj cid.Cid) (bool, error) {
	_, err := cio.Get(ctx, obj)
	if err != nil {
		return false, err
	}
	return true, nil
}
