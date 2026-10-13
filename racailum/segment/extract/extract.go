package extract

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	bstore "github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"

	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

// NewCtx constructs a new extract context
func NewCtx(ctx context.Context, d common.DAL, l *zap.SugaredLogger, aset *actor.Set, latestDealID int64, opts Options, full common.FullNodeApiGetter, eventsBlockstore bstore.Blockstore) (*Ctx, error) {
	ectx := &Ctx{
		C:    ctx,
		D:    d,
		L:    l,
		Opts: opts,
	}

	ectx.Actors.Set = aset
	ectx.LatestDealID = latestDealID
	ectx.FullNode = full
	ectx.EventsBlockstore = eventsBlockstore

	// A fresh, empty state makes the first actor-address tick of this context a
	// full pass. The long-lived extraction path overrides it with the segment's
	// per-process state right after NewCtx (see Segment.extractPart); callers
	// that only dry-run or extract without persisting keep this throwaway state
	// so they never advance the real one. Its part bucket is throwaway too: it
	// is never Committed, so nothing leaks into the real state.
	ectx.ActorAddressState = NewActorAddressWriteState()
	ectx.ActorAddressPart = ectx.ActorAddressState.BeginPart()
	return ectx, nil
}

// Ctx contains all required values and components for extracting chain data
type Ctx struct {
	C    context.Context
	D    common.DAL
	L    *zap.SugaredLogger
	Opts Options

	Actors struct {
		Set *actor.Set
	}

	LatestDealID     int64 // latest dealID of DealProposal
	FullNode         common.FullNodeApiGetter
	EventsBlockstore bstore.Blockstore

	// ActorAddressState makes extractActorAddress emit only the ActorIDs that
	// this process has not written yet instead of re-sending all ~3.84e6 rows
	// every hour. It is per-process on purpose: an empty state (process start)
	// turns the first tick into a full pass, which is also what heals rows a
	// rollback (bell sethead) deleted. See extract/actor_address.go.
	ActorAddressState *ActorAddressWriteState

	// ActorAddressPart is the pending bucket of the part currently being
	// extracted. extractActorAddress files every ActorID it emits under it;
	// Segment.persistPartDocs Commits it once the part's insert landed, or
	// Discards it on failure so the next round re-emits the IDs. Callers that
	// do not persist get a throwaway part from NewCtx and never Commit it.
	// See extract/actor_address.go.
	ActorAddressPart *ActorAddressPart
}

// NewRes constructs a new extract result
func NewRes(docCap, regularStatesCap int) *Res {
	return &Res{
		Docs:          make([]common.Document, 0, docCap),
		RegularStates: make([]*common.ActorHead, 0, regularStatesCap),
	}
}

// Res is a collection of extracted common.Documents
type Res struct {
	RegularStates []*common.ActorHead
	Docs          []common.Document
}

var ActorIDMapping = NewActorIDMap()

type ActorIDMap struct {
	m  map[address.Address]address.Address
	lk sync.RWMutex
}

func NewActorIDMap() *ActorIDMap {
	return &ActorIDMap{m: make(map[address.Address]address.Address)}
}

func (am *ActorIDMap) GetActorID(addr address.Address) (address.Address, bool) {
	am.lk.RLock()
	defer am.lk.RUnlock()

	actorID, ok := am.m[addr]
	return actorID, ok
}

func (am *ActorIDMap) SetActorID(addr, actorID address.Address) {
	am.lk.Lock()
	defer am.lk.Unlock()

	am.m[addr] = actorID
}

func LookupID(ctx *Ctx, addr address.Address, ts *types.TipSet) (address.Address, error) {
	actorID, ok := ActorIDMapping.GetActorID(addr)
	if !ok {
		aID, err := ctx.D.LookupID(ctx.C, addr, ts)
		if err != nil {
			ctx.L.Warnf("failed to lookup actor id: %s,err: %s, fallback to full node", addr, err.Error())
			actorID, err = ctx.FullNode.GetAppropriateAPI().StateLookupID(ctx.C, addr, ts.Key())
			if err != nil {
				return address.Undef, err
			}
		} else {
			actorID, err = address.NewIDAddress(uint64(aID))
			if err != nil {
				return address.Undef, fmt.Errorf("parse actor id(%d) failed: %w", aID, err)
			}
		}

		ActorIDMapping.SetActorID(addr, actorID)
	}

	return actorID, nil
}
