package api

import (
	"context"
)

type BellAPI interface {
	SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error)
	SetSampleRate(ctx context.Context, rate float64) (old float64, err error)
	GetSampleRate(ctx context.Context) (float64, error)
	ShutDown(ctx context.Context) error
}
