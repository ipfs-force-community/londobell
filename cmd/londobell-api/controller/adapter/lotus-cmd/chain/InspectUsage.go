package chain

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs/go-cid"
)

func GetChainInspectUsage(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainInspectUsage")
	req := lotusCmdModel.ChainInspectUsageReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet

	api := adapter.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	cur := ts
	var msgs []lapi.Message
	for i := 0; i < req.Length; i++ {
		pmsgs, err := api.ChainGetParentMessages(ctx, cur.Blocks()[0].Cid())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		msgs = append(msgs, pmsgs...)

		next, err := api.ChainGetTipSet(ctx, cur.Parents())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		cur = next
	}

	codeCache := make(map[address.Address]cid.Cid)

	lookupActorCode := func(a address.Address) (cid.Cid, error) {
		c, ok := codeCache[a]
		if ok {
			return c, nil
		}

		act, err := api.StateGetActor(ctx, a, ts.Key())
		if err != nil {
			return cid.Undef, err
		}

		codeCache[a] = act.Code
		return act.Code, nil
	}

	bySender := make(map[string]int64)
	byDest := make(map[string]int64)
	byMethod := make(map[string]int64)
	bySenderC := make(map[string]int64)
	byDestC := make(map[string]int64)
	byMethodC := make(map[string]int64)

	var sum int64
	for _, m := range msgs {
		bySender[m.Message.From.String()] += m.Message.GasLimit
		bySenderC[m.Message.From.String()]++
		byDest[m.Message.To.String()] += m.Message.GasLimit
		byDestC[m.Message.To.String()]++
		sum += m.Message.GasLimit

		code, err := lookupActorCode(m.Message.To)
		if err != nil {
			if strings.Contains(err.Error(), types.ErrActorNotFound.Error()) {
				continue
			}

			util.ReturnOnErr(c, alog, err)
			return
		}

		mm := filcns.NewActorRegistry().Methods[code][m.Message.Method] // TODO: use remote map

		byMethod[mm.Name] += m.Message.GasLimit
		byMethodC[mm.Name]++
	}

	type keyGasPair struct {
		Key string
		Gas int64
	}

	mapToSortedKvs := func(m map[string]int64) []keyGasPair {
		var vals []keyGasPair
		for k, v := range m {
			vals = append(vals, keyGasPair{
				Key: k,
				Gas: v,
			})
		}
		sort.Slice(vals, func(i, j int) bool {
			return vals[i].Gas > vals[j].Gas
		})
		return vals
	}

	senderVals := mapToSortedKvs(bySender)
	destVals := mapToSortedKvs(byDest)
	methodVals := mapToSortedKvs(byMethod)

	numRes := req.NumResults

	var (
		Senders   = make([]lotusCmdModel.InspectUsage, 0, req.NumResults)
		Receivers = make([]lotusCmdModel.InspectUsage, 0, req.NumResults)
		Methods   = make([]lotusCmdModel.InspectUsage, 0, req.NumResults)
	)
	for i := 0; i < numRes && i < len(senderVals); i++ {
		sv := senderVals[i]
		Senders = append(Senders, lotusCmdModel.InspectUsage{Key: sv.Key, GasLimitRatio: 100 * float64(sv.Gas) / float64(sum), Total: sv.Gas, Count: bySenderC[sv.Key]})
	}
	for i := 0; i < numRes && i < len(destVals); i++ {
		sv := destVals[i]
		Receivers = append(Receivers, lotusCmdModel.InspectUsage{Key: sv.Key, GasLimitRatio: 100 * float64(sv.Gas) / float64(sum), Total: sv.Gas, Count: byDestC[sv.Key]})
	}

	for i := 0; i < numRes && i < len(methodVals); i++ {
		sv := methodVals[i]
		Methods = append(Methods, lotusCmdModel.InspectUsage{Key: sv.Key, GasLimitRatio: 100 * float64(sv.Gas) / float64(sum), Total: sv.Gas, Count: byMethodC[sv.Key]})
	}

	res.Data = lotusCmdModel.ChainInspectUsageRes{
		Epoch:     ts.Height(),
		Senders:   Senders,
		Receivers: Receivers,
		Methods:   Methods,
	}

	c.JSON(http.StatusOK, res)
}
