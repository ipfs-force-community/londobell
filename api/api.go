package api

import (
	"context"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

type BellAPI interface {
	SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error)
	SetSampleRate(ctx context.Context, rate float64) (old float64, err error)
	GetSampleRate(ctx context.Context) (float64, error)
	ShutDown(ctx context.Context) error
}

type MultiAPI interface {
	LoadDBState(url string) (multiquery.DataBaseState, bool, error)
}
