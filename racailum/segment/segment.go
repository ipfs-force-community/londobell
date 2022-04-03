package segment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs-force-community/londobell/metrics"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"

	"go.opencensus.io/stats"
	"go.opencensus.io/trace"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mdict"
	"github.com/ipfs-force-community/londobell/racailum/segment/aggregate"
)

var log = logging.Logger("segment")

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

// New attempts to construct a *Segment
func New(ctx context.Context, name string, opts Options, aggopt aggregate.Options, mgr *Manager, cs common.ChainStore, stm common.StateManager) (*Segment, error) {
	bound, bhas, err := mgr.LoadBoundary(name)
	if err != nil {
		return nil, fmt.Errorf("load boundary: %w", err)
	}

	if !bhas {
		return nil, fmt.Errorf("boundary not found")
	}

	info, ihas, err := mgr.LoadInfo(name)
	if err != nil {
		return nil, fmt.Errorf("load info: %w", err)
	}

	if !ihas {
		return nil, fmt.Errorf("info not found")
	}

	multiWdocs := &mgoutil.MultiDB{}
	dicts := &mdict.Dicts{}

	for _, write := range info.DSN.NewWrite {
		wcli, err := mgoutil.Connect(ctx, write)
		if err != nil {
			return nil, fmt.Errorf("connect to write db: %w", err)
		}

		wdb := wcli.Database(name)

		wdoc, err := mgoutil.NewMgoDocDB(ctx, wcli, wdb)
		if err != nil {
			return nil, fmt.Errorf("construct write doc db: %w", err)
		}

		dict, err := mdict.NewDict(wdb)
		if err != nil {
			return nil, fmt.Errorf("construct Dict: %w", err)
		}

		err = multiWdocs.SetDbs(wdoc)
		if err != nil {
			return nil, fmt.Errorf("multiwdocs setdbs: %w", err)
		}

		err = dicts.SetDicts(dict)
		if err != nil {
			return nil, fmt.Errorf("dicts setdicts: %w", err)
		}
	}

	//rdoc := wdoc
	var rdoc common.DocumentDB
	if info.DSN.Read != "" {
		rcli, err := mgoutil.Connect(ctx, info.DSN.Read)
		if err != nil {
			return nil, fmt.Errorf("connect to read db: %w", err)
		}

		rdoc, err = mgoutil.NewMgoDocDB(ctx, rcli, rcli.Database(name))
		if err != nil {
			return nil, fmt.Errorf("construct read doc db: %w", err)
		}
	}

	agg, err := aggregate.New(aggopt, multiWdocs)
	if err != nil {
		return nil, err
	}

	seg := &Segment{
		name: name,
		opts: opts,
		db:   multiWdocs,
		rdb:  rdoc,
		agg:  agg,
		mgr:  mgr,
	}

	seg.bound.Boundary = bound

	seg.dal.ChainStore = cs
	seg.dal.ChainDict = dicts
	seg.dal.StateManager = stm

	return seg, nil
}

// Segment is one partition of the structrued data extracted from the chain
type Segment struct {
	name string

	opts Options
	db   common.DocumentDB
	rdb  common.DocumentDB
	agg  *aggregate.Aggregator
	mgr  *Manager

	headNotify chan *types.TipSet

	bound struct {
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
	log.Info("start head watching loop start")
	defer log.Info("stop head watching loop")

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
	ctx, span := trace.StartSpan(ctx, "segment.Extract")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(rawts.Height())))
	defer span.End()

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

	if err := s.SaveFinalHeight(ctx, tipsets[len(tipsets)-1]); err != nil {
		return err
	}

	if err := s.updateBoundary(ctx, tipsets[len(tipsets)-1], nil); err != nil {
		return err
	}

	return nil
}

// Aggregate tries to do aggregationg with given tipsets
func (s *Segment) Aggregate(ctx context.Context, tss []*common.LinkedTipSet) error {
	err := s.agg.Aggregate(ctx, tss)
	if err != nil {
		return err
	}

	return nil
}

func (s *Segment) updateBoundary(ctx context.Context, hi, lo *common.LinkedTipSet) error {
	_, span := trace.StartSpan(ctx, "segment.updateBoundary")
	defer span.End()

	prev := s.bound.Boundary

	if hi != nil {
		stats.Record(ctx, metrics.UpperBoundary.M(int64(hi.Height())))
		s.bound.SetHi(hi)
	}

	if lo != nil {
		stats.Record(ctx, metrics.LowerBoundary.M(int64(lo.Height())))
		s.bound.SetLo(lo)
	}

	err := s.mgr.SetBoundary(s.name, s.bound.Boundary)
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

// ReadBoundary returns the boundary of the segment
func (s *Segment) ReadBoundary() Boundary {
	s.bound.RLock()
	b := s.bound.Boundary
	s.bound.RUnlock()
	return b
}

// Name returns the name of the segment
func (s *Segment) Name() string {
	return s.name
}

// ReadDB returns the read only db instance of the segment
func (s *Segment) ReadDB() common.DocumentDB {
	return s.rdb
}

// SaveFinalHeight saves final height
func (s *Segment) SaveFinalHeight(ctx context.Context, hi *common.LinkedTipSet) error {
	elog := log.With("finalHeight", hi.Height())
	elog.Info("save final height")
	res := extract.NewRes(0, 0)
	docs := make([][]common.Document, 1)

	doc, err := model.NewFinalHeight(hi)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, doc)
	docs[0] = res.Docs
	if err := s.insertMany(ctx, elog, docs); err != nil {
		return fmt.Errorf("SaveFinalHeight err: %w", err)
	}

	return nil
}
