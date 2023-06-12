package fullnode

import (
	"context"

	"github.com/filecoin-project/lotus/api/v1api"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
)

var (
	API   *AppropriateAPI
	log   = logging.Logger("appropriate api")
	Fxlog = &fxlogger{
		ZapEventLogger: log,
	}
	Components StateComponents
)

type Candidate struct {
	ts     *types.TipSet
	gap    abi.ChainEpoch
	weight types.BigInt
	api    v1api.FullNode
	url    string
	closer jsonrpc.ClientCloser
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

func GetFullNodeAPI(ctx context.Context, url string) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	api, closer, err := client.NewFullNodeRPCV0(ctx, url, nil)
	if err != nil {
		return nil, nil, err
	}
	return api, closer, nil
}

func InjectAppropriateFullNode(full v1api.FullNode) dix.Option {
	return dix.Override(new(v1api.FullNode), func(lc fx.Lifecycle) v1api.FullNode {
		return full
	})
}
