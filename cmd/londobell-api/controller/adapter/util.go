package adapter

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/buildnet"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
)

var (
	API v0api.FullNode
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

func ChooseAPI(cctx *cli.Context) error {
	urls := cctx.StringSlice("apis")

	candidates := make([]Candidate, 0, len(urls))

	curEpoch := buildnet.GetCurEpoch()
	for _, url := range urls {
		api, err := GetFullNodeAPI(cctx.Context, url)
		if err != nil {
			log.Warnf("api:%v is not accessiable", url)
			continue
		}

		headTs, err := api.ChainHead(cctx.Context)
		if err != nil {
			log.Warnf("api:%v is not accessiable", url)
			continue
		}

		headWeight, err := api.ChainTipSetWeight(cctx.Context, headTs.Key())
		if err != nil {
			log.Warnf("api:%v is not accessiable", url)
			continue
		}

		candidates = append(candidates, Candidate{ts: headTs, gap: curEpoch - headTs.Height(), weight: headWeight, api: api, url: url})
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no available APIs: %v", urls)
	}

	for i := range candidates {
		log.Infof("candidates[%v]: url: %v, ts: %v, ts.Height: %v, curEpoch: %v, weight: %v", i, candidates[i].url, candidates[i].ts, candidates[i].ts.Height(), curEpoch, candidates[i].weight)
	}

	candidate := candidates[0]
	for i := 1; i < len(candidates); i++ {
		// choose unforked node which has more weight
		if types.BigCmp(candidates[i].weight, candidate.weight) > 0 {
			candidate = candidates[i]
		}
	}

	// more appropriate candidate is unsynchronized
	if candidate.gap > 10 {
		return fmt.Errorf("no available APIs: %v", urls)
	}

	API = candidate.api
	log.Infof("choose appropriate api: %v, gap: %v", candidate.url, candidate.gap)
	return nil
}
