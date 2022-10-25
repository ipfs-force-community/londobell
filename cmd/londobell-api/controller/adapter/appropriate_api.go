package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
)

type AppropriateAPI struct {
	api   v0api.FullNode
	urls  []string
	apiMx sync.RWMutex
}

func NewAppropriateAPI(urls []string) *AppropriateAPI {
	return &AppropriateAPI{urls: urls}
}

func (a *AppropriateAPI) GetAppropriateAPI() v0api.FullNode {
	a.apiMx.RLock()
	defer a.apiMx.RUnlock()
	return a.api
}

func (a *AppropriateAPI) SetAppropriateAPI(api v0api.FullNode) {
	a.apiMx.Lock()
	defer a.apiMx.Unlock()
	a.api = api
}

func (a *AppropriateAPI) Choose(ctx context.Context) error {
	a.apiMx.RLock()
	urls := a.urls
	a.apiMx.RUnlock()

	candidates := make([]Candidate, 0, len(urls))

	curEpoch := common.GetCurEpoch()
	for _, url := range urls {
		api, err := GetFullNodeAPI(ctx, url)
		if err != nil {
			log.Warnf("api:%v is not accessiable", url)
			continue
		}

		headTs, err := api.ChainHead(ctx)
		if err != nil {
			log.Warnf("api:%v is not accessiable", url)
			continue
		}

		headWeight, err := api.ChainTipSetWeight(ctx, headTs.Key())
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
		return fmt.Errorf("no available APIs: %v, for gap: %v", urls, candidate.gap)
	}

	previousAPI := a.GetAppropriateAPI()
	if previousAPI != candidate.api {
		a.SetAppropriateAPI(candidate.api)
		log.Infof("choose more appropriate api: %v, gap: %v", candidate.url, candidate.gap)
	} else {
		log.Infof("current api is more appropriate api: %v, gap: %v", candidate.url, candidate.gap)
	}

	return nil
}
