package api

import (
	"context"

	"github.com/filecoin-project/lotus/node/modules/dtypes"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum"
	"github.com/ipfs-force-community/londobell/racailum/segment"
)

var _ BellAPI = (*BellNodeAPI)(nil)
var log = logging.Logger("rpc")

type BellNodeAPI struct {
	fx.In
	Ra         *racailum.RaCailum
	CS         common.ChainStore
	Notifier   common.HeadNotifier
	ShutDownCh dtypes.ShutdownChan
	SegMgr     *segment.Manager
}

func (m *BellNodeAPI) SegmentDetail(ctx context.Context, name string) (*SegmentDetail, error) {
	var res SegmentDetail
	info, has, err := m.SegMgr.LoadInfo(name)
	log.Info(info)
	if err != nil {
		log.Infof("load %s info err: %v", name, err)
		return nil, err
	}
	if has {
		res.Info = &info
	}
	boundary, has, err := m.SegMgr.LoadBoundary(name)
	log.Info(boundary)
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
