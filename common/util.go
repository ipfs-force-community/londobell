package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
)

// IsCtxCanceled checks if an error is caused by context.Canceled
func IsCtxCanceled(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, context.Canceled)
}

// NonCtxCanceledErr returns nil if the error is caused by context.Canceled
func NonCtxCanceledErr(err error) error {
	if IsCtxCanceled(err) {
		return nil
	}

	return err
}

// FormatTipSetEpochRange returns a string shows the lo & hi bound of the given tipsets
func FormatTipSetEpochRange(tss []*LinkedTipSet) string {
	return fmt.Sprintf("[%d, %d]", tss[0].Height(), tss[len(tss)-1].Height())
}

// FormatTipSet returns a compact representation of the tipset
func FormatTipSet(ts *types.TipSet) string {
	return fmt.Sprintf("%s@%d", ts.Key(), ts.Height())
}

// LoadLinkedTipSet attempts to load a tipset with its next
func LoadLinkedTipSet(cs ChainStore, child types.TipSetKey) (*LinkedTipSet, error) {
	childts, err := cs.LoadTipSet(child)
	if err != nil {
		return nil, fmt.Errorf("load child ts: %w", err)
	}

	ts, err := cs.LoadTipSet(childts.Parents())
	if err != nil {
		return nil, fmt.Errorf("load ts: %w", err)
	}

	parentts, err := cs.LoadTipSet(ts.Parents())
	if err != nil {
		return nil, fmt.Errorf("load parent ts: %w", err)
	}

	return &LinkedTipSet{
		TipSet: ts,
		Parent: parentts,
		Child:  childts,
	}, nil
}
