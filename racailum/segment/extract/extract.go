package extract

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/cliex"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

// NewCtx constructs a new extract context
func NewCtx(ctx context.Context, d common.DAL, l *zap.SugaredLogger, aset *actor.Set, latestDealID int64, opts Options, cluster *cliex.Cluster) (*Ctx, error) {
	ectx := &Ctx{
		C:    ctx,
		D:    d,
		L:    l,
		Opts: opts,
	}

	ectx.Actors.Set = aset
	ectx.LatestDealID = latestDealID
	ectx.Cluster = cluster
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

	LatestDealID int64 // latest dealID of DealProposal
	Cluster      *cliex.Cluster
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
	var err error
	actorID, ok := ActorIDMapping.GetActorID(addr)
	if !ok {
		actorID, err = ctx.D.LookupID(ctx.C, addr, ts)
		if err != nil {
			ctx.L.Warnf("failed to lookup actor id: %s,err: %s, fallback to full node", addr, err.Error())
			actorID, err = ctx.Cluster.Current.FullNode.StateLookupID(ctx.C, addr, ts.Key())
			if err != nil {
				return address.Undef, err
			}
		}

		ActorIDMapping.SetActorID(addr, actorID)
	}

	return actorID, nil
}
