package api

import (
	"context"

	segment2 "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"

	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/ipfs-force-community/londobell/racailum/segment"
	"github.com/ipfs-force-community/londobell/racailum/tracing"
)

var _ BellAPI = (*BellNodeAPI)(nil)
var _ MultiAPI = (*MultiNodeAPI)(nil)
var log = logging.Logger("rpc")

type BellNodeAPI struct {
	fx.In
	//Ra         *racailum.RaCailum
	//CS         common.ChainStore
	//Notifier   common.HeadNotifier
	ShutDownCh dtypes.ShutdownChan
	SegMgr     *segment.Manager
}

func (m *BellNodeAPI) SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error) {
	var res SegmentDetail
	info, has, err := m.SegMgr.LoadInfo(name)
	//log.Info(info)
	if err != nil {
		log.Infof("load %s info err: %v", name, err)
		return nil, err
	}
	if has {
		res.Info = &info
	}
	boundary, has, err := m.SegMgr.LoadBoundary(name)
	//log.Info(boundary)
	if err != nil {
		log.Infof("load %s bound err: %v", name, err)
		return nil, err
	}
	if has {
		res.Boundary = &boundary
	}
	active, has, err := m.SegMgr.LoadActive()
	if err != nil {
		log.Infof("load active segment err: %v", err)
		return nil, err
	}
	if has {
		res.Active = active
	}
	return &res, nil
}
func (m *BellNodeAPI) ShutDown(ctx context.Context) error {
	close(m.ShutDownCh)
	return nil
}

func (m *BellNodeAPI) SetSampleRate(ctx context.Context, rate float64) (old float64, err error) {
	return tracing.SetSampleRate(rate)
}
func (m *BellNodeAPI) GetSampleRate(ctx context.Context) (float64, error) {
	return tracing.GetSampleRate()
}

type MultiNodeAPI struct {
	fx.In
	SegMgr *segment2.SegManager
}

func (m *MultiNodeAPI) LoadDBInfo(name string) (segment2.Info, error) {
	info, found, err := m.SegMgr.LoadInfo(name)
	if err != nil {
		return segment2.Info{}, err
	}

	if !found {
		return segment2.Info{}, nil
	}

	return info, nil
}
