package adapter

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
)

var (
	API *AppropriateAPI
	log = logging.Logger("adapter")
)

type Candidate struct {
	ts     *types.TipSet
	gap    abi.ChainEpoch
	weight types.BigInt
	api    v0api.FullNode
	url    string
}

type StateComponents struct {
	fx.In
	SM   common.StateManager
	CS   common.ChainStore
	Full v0api.FullNode
}

type fxlogger struct {
	*logging.ZapEventLogger
}

func (l *fxlogger) Printf(msg string, args ...interface{}) {
	l.ZapEventLogger.Debugf(msg, args...)
}

func GetFullNodeAPI(ctx context.Context, url string) (v0api.FullNode, error) {
	api, _, err := client.NewFullNodeRPCV0(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	return api, nil
}
