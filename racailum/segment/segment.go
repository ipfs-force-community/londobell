package segment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/aggregate"
)

var log = logging.Logger("segment")

// Boundary marks the high & low bound of the segment
type Boundary struct {
	Hi Anchor
	Lo Anchor
}

// SetHi set the high bound to the given tipset
func (b *Boundary) SetHi(ts *common.LinkedTipSet) {
	b.Hi = Anchor{
		Epoch: ts.Height(),
		TSK:   ts.Key(),
		State: ts.State(),
	}
}

// SetLo set the high bound to the given tipset
func (b *Boundary) SetLo(ts *common.LinkedTipSet) {
	b.Lo = Anchor{
		Epoch: ts.Height(),
		TSK:   ts.Key(),
		State: ts.State(),
	}
}

// Anchor contains most significant info about a tipset
type Anchor struct {
	Epoch abi.ChainEpoch
	TSK   types.TipSetKey
	State cid.Cid
}

// Is checks if the anchor is the given tipset
func (a *Anchor) Is(ts *common.LinkedTipSet) bool {
	return a.TSK == ts.Key() && a.State == ts.State()
}

// OptionsKey is the meta key for options
func OptionsKey(name string) string {
	return fmt.Sprintf("seg-%s-options", name)
}

// BoundaryKey is the meta key for boundary
func BoundaryKey(name string) string {
	return fmt.Sprintf("seg-%s-boundary", name)
}

// SetBoundary sets the boundary for a given named segment
func SetBoundary(ctx context.Context, name string, metamgr common.MetaManager, hi, lo *common.LinkedTipSet) error {
	bkey := BoundaryKey(name)
	var b Boundary
	if _, err := metamgr.Load(ctx, bkey, &b); err != nil {
		return fmt.Errorf("load boundary: %w", err)
	}

	if hi != nil {
		b.SetHi(hi)
	}

	if lo != nil {
		b.SetLo(lo)
	}

	if err := metamgr.Update(ctx, bkey, b); err != nil {
		return fmt.Errorf("update boundary: %w", err)
	}

	return nil
}

// New attempts to construct a *Segment
func New(name string, aggopt aggregate.Options, db common.DocumentDB, metamgr common.MetaManager, cs common.ChainStore, dict common.ChainDict, stm common.StateManager, optfns ...OptionFn) (*Segment, error) {
	initCtx := context.Background()

	opts := DefaultOptions()
	optKey := OptionsKey(name)
	if _, err := metamgr.Load(initCtx, optKey, &opts); err != nil {
		return nil, fmt.Errorf("load options: %w", err)
	}

	for _, ofn := range optfns {
		ofn(&opts)
	}

	bkey := BoundaryKey(name)
	var bound Boundary
	loaded, err := metamgr.Load(initCtx, bkey, &bound)
	if err != nil {
		return nil, fmt.Errorf("load boundary: %w", err)
	}

	if !loaded {
		return nil, fmt.Errorf("boundady is required")
	}

	agg, err := aggregate.New(aggopt, db)
	if err != nil {
		return nil, err
	}

	seg := &Segment{
		name:    name,
		opts:    opts,
		db:      db,
		agg:     agg,
		metamgr: metamgr,
	}

	seg.bound.key = bkey
	seg.bound.Boundary = bound

	seg.dal.ChainStore = cs
	seg.dal.ChainDict = dict
	seg.dal.StateManager = stm

	return seg, nil
}

// Segment is one partition of the structrued data extracted from the chain
type Segment struct {
	name string

	opts    Options
	db      common.DocumentDB
	metamgr common.MetaManager
	agg     *aggregate.Aggregator

	headNotify chan *types.TipSet

	bound struct {
		key string

		sync.RWMutex
		Boundary
	}

	dal struct {
		common.ChainStore
		common.ChainDict
		common.StateManager
	}
}

// Incoming responds to a new heaviest tipset
func (s *Segment) Incoming(ctx context.Context, ts *types.TipSet) {
	select {
	case s.headNotify <- ts:

	default:

	}

	return

}

// Run starts a watching loop for incoming tipsets
func (s *Segment) Run(ctx context.Context) {
	log.Info("start head wathcing loop start")
	defer log.Info("stop head wathcing loop")

	for {
		select {
		case <-ctx.Done():
			return

		case ts := <-s.headNotify:
			if err := s.Extract(ctx, ts); err != nil {
				log.Errorw("failed to handle inocoming tipset", "tsk", ts.Key(), "tsh", ts.Height(), "err", err.Error())
			}
		}
	}
}

// Extract attempts to extract data between given tipset and the hi-bound of the segment
func (s *Segment) Extract(ctx context.Context, rawts *types.TipSet) error {
	tsk := rawts.Key()
	tsh := rawts.Height()

	start := time.Now()
	defer func() {
		log.Infow("extract done", "raw-tsk", tsk, "raw-tsh", tsh, "elapsed", time.Now().Sub(start).String())
	}()

	s.bound.Lock()
	defer s.bound.Unlock()

	lo := s.bound.Lo.Epoch
	if tsh <= lo {
		return fmt.Errorf("%s is not belong to this segment", common.FormatTipSet(rawts))
	}

	hi := s.bound.Hi.Epoch
	if tsh > hi+s.opts.Extract.MaxBackward {
		return fmt.Errorf("%s is too far away from current upper bound @%d", common.FormatTipSet(rawts), hi)
	}

	if hi+s.opts.Extract.MinPeriod+s.opts.Extract.Confidence >= tsh {
		return nil
	}

	tipsets, err := ExtractLinkedTipSets(s.dal.ChainStore, rawts, &hi)
	if err != nil {
		return err
	}

	if !s.bound.Hi.Is(tipsets[0]) {
		return fmt.Errorf("current segment is not the ancestor of the incoming chain: %s", tipsets[0])
	}

	for i := 1; i < len(tipsets); i++ {
		tipsets[i].Parent = tipsets[i-1].TipSet
	}

	tipsets = tipsets[1:]

	confidentEpoch := tsh - s.opts.Extract.Confidence
	tssize := len(tipsets)
	for ; tssize > 0; tssize-- {
		if tipsets[tssize-1].Height() <= confidentEpoch {
			break
		}
	}

	tipsets = tipsets[:tssize]
	if len(tipsets) == 0 {
		return nil
	}

	if err := s.extractTipSets(ctx, tipsets); err != nil {
		return err
	}

	if err := s.Aggregate(ctx, tipsets); err != nil {
		return err
	}

	if err := s.updateBoundary(ctx, tipsets[len(tipsets)-1], nil); err != nil {
		return err
	}

	return nil
}

// Aggregate tries to do aggregationg with given tipsets
func (s *Segment) Aggregate(ctx context.Context, tss []*common.LinkedTipSet) error {
	return s.agg.Aggregate(ctx, tss)
}

func (s *Segment) updateBoundary(ctx context.Context, hi, lo *common.LinkedTipSet) error {
	prev := s.bound.Boundary

	if hi != nil {
		s.bound.SetHi(hi)
	}

	if lo != nil {
		s.bound.SetLo(lo)
	}

	err := s.metamgr.Update(ctx, s.bound.key, s.bound.Boundary)
	if err == nil {
		return nil
	}

	s.bound.Boundary = prev
	return err
}

// SetBoundary updates the boundary of the segment
func (s *Segment) SetBoundary(ctx context.Context, hi, lo *common.LinkedTipSet) error {
	s.bound.Lock()
	defer s.bound.Unlock()

	// TODO: reset, cleanup and other stuffs
	return s.updateBoundary(ctx, hi, lo)
}
