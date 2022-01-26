package segment

import (
	"context"
	"sync"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opencensus.io/stats"

	"github.com/ipfs-force-community/londobell/metrics"
)

type SegProxy struct {
	// range[start, end]
	mux  sync.Mutex
	segs []*Segment
}

func (sp *SegProxy) Distribute(ctx context.Context, ts *types.TipSet) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	elog := log.With("tipset", ts.Height())
	elog.Info("tipset to distribute")
	for _, seg := range sp.segs {
		boundary := seg.ReadBoundary()
		if ts.Height() >= boundary.Lo.Epoch && ts.Height() < boundary.Hi.Epoch {
			seg.headNotify <- ts
			elog.Infof("distribute tipset at epoch: %s to segment: %s", ts.Height(), seg.Name())
		}
	}
	elog.Warn("no matching segment to tipset")

	stats.Record(ctx, metrics.MissedTipsetCnt.M(1))
}

func (sp *SegProxy) Register(ctx context.Context, seg *Segment) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	sp.segs = append(sp.segs, seg)
	go seg.Run(ctx)
}
