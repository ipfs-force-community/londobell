package chainx

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ipld/go-car"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
)

// Genesis returns the genesis block header
func Genesis() (*types.BlockHeader, error) {
	genbytes := build.MaybeGenesis()

	cr, err := car.NewCarReader(bytes.NewBuffer(genbytes))
	if err != nil {
		return nil, err
	}

	for {
		blk, err := cr.Next()
		switch err {
		case io.EOF:
			return nil, fmt.Errorf("root block %s not found", cr.Header.Roots[0])

		case nil:
			if blk.Cid() == cr.Header.Roots[0] {
				return types.DecodeBlock(blk.RawData())
			}

		default:
			return nil, err
		}
	}

}
