package fullnode

import (
	"context"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/londobell/common"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

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
	api    v0api.FullNode
	url    string
	closer jsonrpc.ClientCloser
}

type StateComponents struct {
	fx.In
	SM   common.StateManager
	CS   common.ChainStore
	Full common.FullNodeApiGetter
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

func InjectAppropriateFullNode(full v0api.FullNode) dix.Option {
	return dix.Override(new(v0api.FullNode), func(lc fx.Lifecycle) v0api.FullNode {
		return full
	})
}

func InjectFullNodeApiGetter() dix.Option {
	return dix.Override(new(common.FullNodeApiGetter), func(lc fx.Lifecycle) common.FullNodeApiGetter {
		return API
	})
}
