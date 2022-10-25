package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/buildnet"
)

var (
	Loc, _      = time.LoadLocation("Asia/Shanghai")
	BaseTime, _ = time.Parse(time.RFC3339, buildnet.BeginTime)
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
	childts, err := cs.LoadTipSet(context.Background(), child)
	if err != nil {
		return nil, fmt.Errorf("load child ts: %w", err)
	}

	ts, err := cs.LoadTipSet(context.Background(), childts.Parents())
	if err != nil {
		return nil, fmt.Errorf("load ts: %w", err)
	}

	parentts, err := cs.LoadTipSet(context.Background(), ts.Parents())
	if err != nil {
		return nil, fmt.Errorf("load parent ts: %w", err)
	}

	return &LinkedTipSet{
		TipSet: ts,
		Parent: parentts,
		Child:  childts,
	}, nil
}

func IsZeroHour(curEpoch abi.ChainEpoch) bool {
	curTime := time.Unix(BaseTime.Unix()+int64(curEpoch)*30, 0).In(Loc)
	if curTime.Hour() == 0 && curTime.Minute() == 0 && curTime.Second() == 0 {
		return true
	}

	return false
}

func CalcTimeByEpoch(height uint64) time.Time {
	return time.Unix(BaseTime.Unix()+int64(height)*30, 0).In(Loc)
}

func GetCurEpoch() abi.ChainEpoch {
	return abi.ChainEpoch((time.Now().Unix() - BaseTime.Unix()) / 30)
}

func AddAddressPrefix(addr string) string {
	return buildnet.NetPrefix + addr
}
