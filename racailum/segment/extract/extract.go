package extract

import (
	"context"

	"go.uber.org/zap"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/actor"
)

// NewCtx constructs a new extract context
func NewCtx(ctx context.Context, d common.DAL, l *zap.SugaredLogger, aset *actor.Set, opts Options) (*Ctx, error) {
	ectx := &Ctx{
		C:    ctx,
		D:    d,
		L:    l,
		Opts: opts,
	}

	ectx.Actors.Set = aset
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
