package adapter

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"

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

func GetFullNodeAPI(ctx context.Context, url string) (v0api.FullNode, error) {
	api, _, err := client.NewFullNodeRPCV0(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	return api, nil
}
