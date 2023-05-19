package segment

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

// ExtractLinkedTipSets extracts linked tipsets within range [lower, from); range [lower, from] if tmp is true
func ExtractLinkedTipSets(cs common.ChainStore, from *types.TipSet, lower *abi.ChainEpoch, tmp bool) ([]*common.LinkedTipSet, error) {
	var destEpoch abi.ChainEpoch

	if lower != nil {
		destEpoch = *lower
	}

	h := from.Height()
	if h <= destEpoch {
		return nil, nil
	}

	length := int(h - destEpoch)
	if tmp {
		length = int(h-destEpoch) + 1
	}
	tss := make([]*common.LinkedTipSet, 0, length)
	var prev *types.TipSet
	_, err := TraverseTipSets(cs, from, func(walked *types.TipSet, walkedEpoch abi.ChainEpoch) (bool, error) {
		if walkedEpoch < destEpoch {
			return false, nil
		}

		if tmp {
			// allow nil child
			tss = append(tss, &common.LinkedTipSet{
				TipSet: walked,
				Child:  prev,
			})
		} else {
			if prev != nil {
				tss = append(tss, &common.LinkedTipSet{
					TipSet: walked,
					Child:  prev,
				})
			}
		}

		prev = walked

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	got := len(tss)
	for i := 0; i < got/2; i++ {
		j := got - i - 1
		tss[i], tss[j] = tss[j], tss[i]
	}

	return tss, nil
}

func TraverseTipSets(cs common.ChainStore, curts *types.TipSet, traverseFn func(*types.TipSet, abi.ChainEpoch) (bool, error)) (int, error) {
	count := 0

	for {
		curh := curts.Height()
		keep, err := traverseFn(curts, curh)
		count++
		if err != nil {
			return count, err
		}

		if !keep || curh == 0 {
			return count, nil
		}

		parentTSK := curts.Parents()
		parentTS, err := cs.LoadTipSet(context.Background(), parentTSK)
		if err != nil {
			return count, err
		}

		curts = parentTS
	}
}
