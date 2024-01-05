package tmpbell

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs-force-community/londobell/metrics"
	"go.opencensus.io/stats"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/racailum/segment"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/vm"
)

type TmpBell struct {
	Full      v0api.FullNode
	activeSeg *segment.Segment
}

var (
	log = logging.Logger("tmp")

	errNoAncestor = errors.New("no ancestor")
	errNullTipSet = errors.New("null tipSet")
	errZeroHeight = errors.New("zero height")

	continuousNullTipSet = 0
)

func New(ctx context.Context, cfg racailum.Config, cs common.ChainStore, stm common.StateManager, segmgr *segment.Manager, full v0api.FullNode) (*TmpBell, error) {
	vm.EnableDetailedTracing = cfg.EnableTracing

	activeSegName, has, err := segmgr.LoadActive()
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, fmt.Errorf("tmp: no active segment")
	}

	activeSeg, err := segment.New(ctx, activeSegName, cfg.Segment, cfg.Aggregate, segmgr, cs, stm, full)
	if err != nil {
		return nil, err
	}

	tmp := &TmpBell{
		Full:      full,
		activeSeg: activeSeg,
	}
	return tmp, nil
}

// MonitorForTmpDB monitors chainHead of lotus and tmpFinalHeight, and decides whether to extract to temporary db
func (t *TmpBell) MonitorForTmpDB(ctx context.Context, tempDBCapacity, triggerSpan uint) error {
	head, err := t.Full.ChainHead(ctx)
	if err != nil {
		return err
	}

	tmpFinalHeight := t.GetTmpFinalHeight()

	if head.Height() < tmpFinalHeight {
		return fmt.Errorf("headHeight %v lags behind tmpFinalHeight %v, maybe lotus had been thinned", head.Height(), tmpFinalHeight)
	}

	log.Infof("headHeight: %v, tmpFinalHeight: %v", head.Height(), tmpFinalHeight)

	// begin temporary db when tmpFinalHeight is closed to headHeight
	if uint(head.Height()-tmpFinalHeight) <= triggerSpan {
		log.Infow("activate tmp db successfully", "headHeight", head.Height(), "tmpFinalHeight", tmpFinalHeight)
		go t.ExtractIncomingHead(ctx, tempDBCapacity)
	} else {
		return fmt.Errorf("activate tmp db failed, headHeight: %v, tmpFinalHeight: %v", head.Height(), tmpFinalHeight)
	}

	return nil
}

func (t *TmpBell) GetTmpFinalHeight() abi.ChainEpoch {
	return t.activeSeg.ReadBoundary().Hi.Epoch
}

func (t *TmpBell) AlertOutdatedFinalHeight(ctx context.Context, outdatedGap int64) {
	tick := time.NewTicker(2 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			finalHeight := t.GetTmpFinalHeight()
			curEpoch := common.GetCurEpoch()
			if curEpoch-finalHeight >= abi.ChainEpoch(outdatedGap) {
				log.Warnf("finalHeight %v lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
				stats.Record(ctx, metrics.OutdatedFinalHeight.M(1))
				continue
			}

			log.Infof("finalHeight %v not lag behind curEpoch %v more than outdatedGap %v", finalHeight, curEpoch, outdatedGap)
			stats.Record(ctx, metrics.OutdatedFinalHeight.M(0))
		case <-ctx.Done():
			return
		}
	}
}
