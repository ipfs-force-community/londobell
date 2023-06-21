package fullnode

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/filecoin-project/lotus/api/client"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
)

type AppropriateAPI struct {
	apiMx    sync.RWMutex
	nodes    []util.Node
	node     Node
	lastNode Node
}

type Node struct {
	url    string
	api    v0api.FullNode
	closer jsonrpc.ClientCloser
}

func NewAppropriateAPI(nodes []util.Node) *AppropriateAPI {
	return &AppropriateAPI{nodes: nodes, node: Node{}, lastNode: Node{}}
}

func (a *AppropriateAPI) GetAppropriateAPI() v0api.FullNode {
	a.apiMx.RLock()
	defer a.apiMx.RUnlock()
	return a.node.api
}

func (a *AppropriateAPI) GetAppropriateUrl() string {
	a.apiMx.RLock()
	defer a.apiMx.RUnlock()
	return a.node.url
}

func (a *AppropriateAPI) GetAppropriateNode() Node {
	a.apiMx.RLock()
	defer a.apiMx.RUnlock()
	return a.node
}

func (a *AppropriateAPI) GetLastAppropriateNode() Node {
	a.apiMx.RLock()
	defer a.apiMx.RUnlock()
	return a.lastNode
}

func (a *AppropriateAPI) SetAppropriateAPI(api v0api.FullNode, url string, closer jsonrpc.ClientCloser) {
	a.apiMx.Lock()
	defer a.apiMx.Unlock()
	a.node.api = api
	a.node.url = url
	a.node.closer = closer
}

func (a *AppropriateAPI) SetLastAppropriateAPI(api v0api.FullNode, url string, closer jsonrpc.ClientCloser) {
	a.apiMx.Lock()
	defer a.apiMx.Unlock()
	a.lastNode.api = api
	a.lastNode.url = url
	a.lastNode.closer = closer
}

func (a *AppropriateAPI) Choose(ctx context.Context) error {
	a.apiMx.RLock()
	nodes := a.nodes
	a.apiMx.RUnlock()

	candidates := make([]Candidate, 0, len(nodes))

	curEpoch := common.GetCurEpoch()
	for _, node := range nodes {
		var requestHeader http.Header
		token := node.Token
		url := node.URL
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
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

		candidates = append(candidates, Candidate{ts: headTs, gap: curEpoch - headTs.Height(), weight: headWeight, api: api, url: url, closer: closer})
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no available APIs: %v", nodes)
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

	// close all inappropriate nodes
	if candidate.gap > 10 {
		log.Warnf("gap %v of candidate %v more than 10", candidate.gap, candidate.url)
	}

	nonCandidates := make([]Candidate, 0, len(candidates)-1)
	for _, c := range candidates {
		if c.url == candidate.url {
			continue
		}
		nonCandidates = append(nonCandidates, c)
	}

	lastNode := a.GetAppropriateNode()

	// record lastNode and current node
	if lastNode.api == nil {
		a.SetAppropriateAPI(candidate.api, candidate.url, candidate.closer)
		closeNonCandidate(nonCandidates)
	} else {
		if lastNode.url != candidate.url {
			a.SetLastAppropriateAPI(lastNode.api, lastNode.url, lastNode.closer)
			a.SetAppropriateAPI(candidate.api, candidate.url, candidate.closer)
			closeNonCandidate(nonCandidates)
		} else {
			a.SetLastAppropriateAPI(lastNode.api, lastNode.url, lastNode.closer)
			a.SetAppropriateAPI(lastNode.api, lastNode.url, lastNode.closer)
			closeNonCandidate(candidates)
		}
	}

	log.Infof("current api is more appropriate api: %v, gap: %v", candidate.url, candidate.gap)

	return nil
}

func closeNonCandidate(nonCandidates []Candidate) {
	for _, nonCandidate := range nonCandidates {
		nonCandidate.closer()
	}
}

func (a *AppropriateAPI) InjectNewFullNode(cctx *cli.Context) (bool, error) {
	appropriateNode := a.GetAppropriateNode()
	lastAppropriateNode := a.GetLastAppropriateNode()

	if lastAppropriateNode.api != nil && appropriateNode.url == lastAppropriateNode.url {
		return false, nil
	}

	if lastAppropriateNode.api != nil && appropriateNode.url != lastAppropriateNode.url {
		// stop app
		err := util.GetStopFuncByUrl(lastAppropriateNode.url)(context.TODO())
		if err != nil {
			log.Errorf("stop app failed: %v, url: %v", err, lastAppropriateNode.url)
			return true, err
		}
		log.Infof("stop app successfully, url: %v", lastAppropriateNode.url)

		// close last fullnode
		lastAppropriateNode.closer()
	}

	// inject new fullnode
	stopFunc, err := dix.New(context.Background(), dep.Bell(context.Background(), Fxlog, &Components), dep.InjectRepoPath(cctx), InjectAppropriateFullNode(appropriateNode.api))
	if err != nil {
		log.Errorf("inject dependencies failed: %v", err)
		return true, err
	}

	util.RegistryStopFuncMap(appropriateNode.url, stopFunc)

	return true, nil
}
