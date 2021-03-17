package grafana

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

var log = logging.Logger("grafana")

var (
	_ http.Handler = (*Grafana)(nil)
)

func getCollections() []string {
	m := schema.Models()
	colMap := map[string]struct{}{}
	for mi := range m {
		colMap[m[mi].D.CollectionName()] = struct{}{}
	}

	cols := make([]string, 0, len(colMap))
	for c := range colMap {
		cols = append(cols, c)
	}

	sort.Strings(cols)
	return cols
}

func New(g *types.BlockHeader) (*Grafana, error) {
	return &Grafana{
		genesis: g,
		cols:    getCollections(),
	}, nil
}

// Grafana is a simple json data source implementation
type Grafana struct {
	genesis *types.BlockHeader
	cols    []string
}

func (g *Grafana) time2epoch(t time.Time) abi.ChainEpoch {
	return abi.ChainEpoch((uint64(t.Unix()) - g.genesis.Timestamp) / builtin.EpochDurationSeconds)
}

func (g *Grafana) epoch2time(h abi.ChainEpoch) time.Time {
	return time.Unix(int64(g.genesis.Timestamp)+int64(h)*builtin.EpochDurationSeconds, 0)
}

func (g *Grafana) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method == http.MethodOptions {
		return
	}

	log.Infow("request", "method", r.Method, "path", r.URL.Path)

	switch r.URL.Path {
	case "/search":
		var req searchReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Errorf("decode: %s", err)
			return
		}

		resp, err := g.serveSearch(&req)
		if err != nil {
			log.Errorf("call: %s", err)
			return
		}

		json.NewEncoder(rw).Encode(resp)
		return

	case "/query":

	case "/annotations":

	case "/tag-keys":

	case "/tag-values":

	}
}

func (g *Grafana) serveSearch(req *searchReq) (searchResp, error) {
	if req.Target != "" {
		return nil, nil
	}

	return g.cols, nil
}
