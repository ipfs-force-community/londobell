package api

import (
	"context"

	segment2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
)

type BellAPI interface {
	SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error)
	SetSampleRate(ctx context.Context, rate float64) (old float64, err error)
	GetSampleRate(ctx context.Context) (float64, error)
	ShutDown(ctx context.Context) error
}

type MultiAPI interface {
	LoadDBInfo(name string) (segment2.Info, error)
}
