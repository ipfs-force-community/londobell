package tmpbell

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

func LoadLinkedTipSet(ctx context.Context, tsKey types.TipSetKey, full common.FullNodeApiGetter) (*common.LinkedTipSet, error) {
	ts, err := full.GetAppropriateAPI().ChainGetTipSet(ctx, tsKey)
	if err != nil {
		return nil, err
	}
	parent, err := full.GetAppropriateAPI().ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return nil, err
	}

	return &common.LinkedTipSet{
		TipSet: ts,
		Child:  nil,
		Parent: parent,
	}, nil
}
