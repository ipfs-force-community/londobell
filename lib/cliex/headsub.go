package cliex

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/common"
)

const (
	minReListenInterval = time.Second
	maxReListenInterval = 10 * time.Second

	nonChanModeInterval = 10 * time.Second
)

var log = logging.Logger("headsub")

func NewHeadSub(full common.FullNodeApiGetter) (*HeadSub, error) {
	return &HeadSub{
		full:     full,
		interval: minReListenInterval,
	}, nil
}

type HeadSub struct {
	full     common.FullNodeApiGetter
	interval time.Duration
}

func (h *HeadSub) Sub(ctx context.Context) (<-chan types.TipSetKey, error) {
	ch := make(chan types.TipSetKey, 1)
	go h.watch(ctx, ch)
	return ch, nil
}

func (h *HeadSub) watch(ctx context.Context, tx chan types.TipSetKey) {
	log.Info("head change loop start")
	defer log.Info("head change loop stop")

	for {
		ch, err := h.reListen(ctx)
		if err != nil {
			log.Errorf("failed to listen head change: %s", err)
			return
		}

		cancel := context.CancelFunc(func() {})

	CHANGES_LOOP:
		for {
			select {
			case <-ctx.Done():
				cancel()
				return

			case changes, ok := <-ch:
				if !ok {
					break CHANGES_LOOP
				}

				cancel()

				applyCtx, applyCancel := context.WithCancel(ctx)
				cancel = applyCancel
				h.applyChanges(applyCtx, tx, changes)
			}
		}

		cancel()
	}
}

func (h *HeadSub) applyChanges(ctx context.Context, tx chan types.TipSetKey, changes []*api.HeadChange) {
	idx := -1
	for i := range changes {
		switch changes[i].Type {
		case store.HCCurrent, store.HCApply:
			idx = i
		}
	}

	if idx == -1 {
		return
	}

	tsk := changes[idx].Val.Key()
	go delaySend(ctx, tx, tsk)
}

func (h *HeadSub) reListen(ctx context.Context) (<-chan []*api.HeadChange, error) {
	for {
		ch, err := h.full.GetAppropriateAPI().ChainNotify(ctx)
		if err == nil {
			h.interval = minReListenInterval
			return ch, nil
		}

		// we have error here, try use non-chan mode
		ch, err = h.reListenInNonChan(ctx)
		if err == nil {
			h.interval = minReListenInterval
			return ch, nil
		}

		log.Errorf("call CahinNotify: %s, will re-call in %s", err, h.interval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-time.After(h.interval):
			h.interval *= 2
			if h.interval > maxReListenInterval {
				h.interval = maxReListenInterval
			}

		}
	}
}

func (h *HeadSub) reListenInNonChan(ctx context.Context) (<-chan []*api.HeadChange, error) {
	tryCtx, tryCancel := context.WithTimeout(ctx, 5*time.Second)
	defer tryCancel()

	head, err := h.full.GetAppropriateAPI().ChainHead(tryCtx)
	if err != nil {
		return nil, err
	}

	ch := make(chan []*api.HeadChange, 16)
	ch <- []*api.HeadChange{
		{
			Type: store.HCCurrent,
			Val:  head,
		},
	}

	go h.startChainHeadLoop(ctx, ch)

	return ch, nil
}

func (h *HeadSub) startChainHeadLoop(ctx context.Context, ch chan []*api.HeadChange) {
	log.Warn("ChainNotify not supportted, use ChainHead instead")
	defer func() {
		log.Warn("ChainHead loop stop")
		close(ch)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(nonChanModeInterval):

		}

		reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
		head, err := h.full.GetAppropriateAPI().ChainHead(reqCtx)
		reqCancel()
		if err != nil {
			log.Errorf("call ChainHead: %s", err)
			return
		}

		select {
		case <-ctx.Done():
			return

		case ch <- []*api.HeadChange{{
			Type: store.HCCurrent,
			Val:  head,
		}}:

		}
	}
}

func delaySend(ctx context.Context, ch chan types.TipSetKey, tsk types.TipSetKey) {
	slog := log.With("tsk", tsk)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")
		return

	case <-timer.C:

	}

	wait := time.NewTimer(time.Second)
	defer wait.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")

	case ch <- tsk:
		slog.Debug("sent")

	case <-wait.C:
		slog.Debug("out chan is full")
	}
}
