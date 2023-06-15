package tmpbell

import (
	"context"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

func LoadLinkedTipSet(ctx context.Context, tsKey types.TipSetKey, full v0api.FullNode) (*common.LinkedTipSet, error) {
	ts, err := full.ChainGetTipSet(ctx, tsKey)
	if err != nil {
		return nil, err
	}
	parent, err := full.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return nil, err
	}

	return &common.LinkedTipSet{
		TipSet: ts,
		Child:  nil,
		Parent: parent,
	}, nil
}
