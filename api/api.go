package api

import (
	"context"
)

type BellAPI interface {
	SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error)
	ShutDown(ctx context.Context) error
}
