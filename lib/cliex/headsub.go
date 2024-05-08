package cliex

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/ipfs-force-community/londobell/common"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
)

const (
	minReListenInterval = time.Second
	maxReListenInterval = 10 * time.Second

	nonChanModeInterval = 10 * time.Second
)

var log = logging.Logger("headsub")

type Node struct {
	API      string
	FullNode v0api.FullNode
	Closer   jsonrpc.ClientCloser
	Gap      abi.ChainEpoch
	Token    string
}
type Cluster struct {
	Current        *Node
	Master         string
	Nodes          []*Node
	MasterGapLimit abi.ChainEpoch
}

func NewHeadSub(cluster *Cluster) (*HeadSub, error) {
	return &HeadSub{
		cluster:  cluster,
		interval: minReListenInterval,
	}, nil
}

type HeadSub struct {
	cluster  *Cluster
	interval time.Duration
}

func (h *HeadSub) Sub(ctx context.Context) (<-chan types.TipSetKey, error) {
	ch := make(chan types.TipSetKey, 1)
	go h.watch(ctx, ch)
	return ch, nil
}

func (h *HeadSub) healthCheck() {
	log.Info("health check start")
	defer log.Info("health check stop")
	var best = h.cluster.Current

	for index, node := range h.cluster.Nodes {

		node, err := InjectFullNode(node.API, node.Token)
		if err != nil {
			log.Warnf("node check failed,node: %s err: %s", node.API, err.Error())
			continue
		}
		h.cluster.Nodes[index] = node
		if node.API == h.cluster.Master {
			if node.Gap < h.cluster.MasterGapLimit {
				log.Info("master check success,node: ", h.cluster.Master)
				// h.cluster.Current = node
				best = node
				break
			}
		}
		if node.Gap < best.Gap {
			best = node
		}

	}
	h.cluster.Current = best
	for _, node := range h.cluster.Nodes {

		if node.API != h.cluster.Current.API {
			if node.Closer != nil {
				node.Closer()
				node.Closer = nil
			}
		}
	}
}

func InjectFullNode(api, token string) (*Node, error) {

	var requestHeader http.Header
	node := Node{
		API: api,
	}
	if token != "" {
		requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
	}

	full, closer, err := client.NewFullNodeRPCV0(context.Background(), api, requestHeader)
	if err != nil {
		return &node, err
	}

	node.FullNode = full
	node.Closer = closer
	node.Gap, err = getGap(&node)
	if err != nil {
		return &node, err
	}
	return &node, nil

}

func getGap(node *Node) (abi.ChainEpoch, error) {
	fmt.Println(node)
	headTipset, err := node.FullNode.ChainHead(context.Background())
	if err != nil {
		return 0, err
	}
	gap := headTipset.Height() - common.GetCurEpoch()

	randomNumber := rand.Intn(5)
	fmt.Println("random number ", randomNumber)
	return gap + abi.ChainEpoch(randomNumber), nil

}

func (h *HeadSub) watch(ctx context.Context, tx chan types.TipSetKey) {
	log.Info("head change loop start")
	defer log.Info("head change loop stop")

	for {
		if len(h.cluster.Nodes) > 1 {
			h.healthCheck()
		}
		ch, err := h.reListen(ctx)
		if err != nil {
			log.Errorf("failed to listen head change: %s", err)
			return
		}

		cancel := context.CancelFunc(func() {})

	CHANGES_LOOP:
		for {
			select {
			case <-ctx.Done():
				cancel()
				return

			case changes, ok := <-ch:
				if !ok {
					break CHANGES_LOOP
				}

				cancel()

				applyCtx, applyCancel := context.WithCancel(ctx)
				cancel = applyCancel
				h.applyChanges(applyCtx, tx, changes)
			}
		}

		cancel()
	}
}

func (h *HeadSub) applyChanges(ctx context.Context, tx chan types.TipSetKey, changes []*api.HeadChange) {
	idx := -1
	for i := range changes {
		switch changes[i].Type {
		case store.HCCurrent, store.HCApply:
			idx = i
		}
	}

	if idx == -1 {
		return
	}

	tsk := changes[idx].Val.Key()
	go delaySend(ctx, tx, tsk)
}

func (h *HeadSub) reListen(ctx context.Context) (<-chan []*api.HeadChange, error) {
	for {
		// check full delay
		// h.full =
		ch, err := h.cluster.Current.FullNode.ChainNotify(ctx)
		if err == nil {
			h.interval = minReListenInterval
			return ch, nil
		}

		// we have error here, try use non-chan mode
		ch, err = h.reListenInNonChan(ctx)
		if err == nil {
			h.interval = minReListenInterval
			return ch, nil
		}

		log.Errorf("call CahinNotify: %s, will re-call in %s", err, h.interval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-time.After(h.interval):
			h.interval *= 2
			if h.interval > maxReListenInterval {
				h.interval = maxReListenInterval
			}

		}
	}
}

func (h *HeadSub) reListenInNonChan(ctx context.Context) (<-chan []*api.HeadChange, error) {
	tryCtx, tryCancel := context.WithTimeout(ctx, 5*time.Second)
	defer tryCancel()

	head, err := h.cluster.Current.FullNode.ChainHead(tryCtx)
	if err != nil {
		return nil, err
	}

	ch := make(chan []*api.HeadChange, 16)
	ch <- []*api.HeadChange{
		{
			Type: store.HCCurrent,
			Val:  head,
		},
	}

	go h.startChainHeadLoop(ctx, ch)

	return ch, nil
}

func (h *HeadSub) startChainHeadLoop(ctx context.Context, ch chan []*api.HeadChange) {
	log.Warn("ChainNotify not supportted, use ChainHead instead")
	defer func() {
		log.Warn("ChainHead loop stop")
		close(ch)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(nonChanModeInterval):

		}

		reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
		head, err := h.cluster.Current.FullNode.ChainHead(reqCtx)
		reqCancel()
		if err != nil {
			log.Errorf("call ChainHead: %s", err)
			return
		}

		select {
		case <-ctx.Done():
			return

		case ch <- []*api.HeadChange{{
			Type: store.HCCurrent,
			Val:  head,
		}}:

		}
	}
}

func delaySend(ctx context.Context, ch chan types.TipSetKey, tsk types.TipSetKey) {
	slog := log.With("tsk", tsk)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")
		return

	case <-timer.C:

	}

	wait := time.NewTimer(time.Second)
	defer wait.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")

	case ch <- tsk:
		slog.Debug("sent")

	case <-wait.C:
		slog.Debug("out chan is full")
	}
}
