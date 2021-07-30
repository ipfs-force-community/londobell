package tipset

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/ipfs/go-cid"

	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"

	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"

	"github.com/filecoin-project/lotus/api"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mir"
	"github.com/dtynn/londobell/racailum/segment/actor"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	execTraceExample := &model.ExecTrace{}
	execTraceExample.Msg.Method = 2
	execTraceExample.Detail.Return = &multisig3.ProposeReturn{}

	schema.Register(
		schema.Model{
			Name: "tipset",
			D:    &model.TipSet{},
		},
		schema.Model{
			Name: "block-header",
			D:    &model.BlockHeader{},
		},
		schema.Model{
			Name: "message: miner.PreCommitSector v2",
			D: &model.Message{
				Detail: model.MessageDetail{
					Actor:  "fil/2/storageminer",
					Method: "PreCommitSector",
					Params: &miner2.PreCommitSectorParams{},
				},
			},
		},
		schema.Model{
			Name: "message: miner.PreCommitSector v3",
			D: &model.Message{
				Detail: model.MessageDetail{
					Actor:  "fil/3/storageminer",
					Method: "PreCommitSector",
					Params: &miner3.PreCommitSectorParams{},
				},
			},
		},
		schema.Model{
			Name: "message: multisig.Propose v2",
			D: &model.Message{
				Detail: model.MessageDetail{
					Actor:  "fil/2/multisig",
					Method: "Propose",
					Params: &multisig2.ProposeParams{},
				},
			},
		},
		schema.Model{
			Name: "message: multisig.Propose v3",
			D: &model.Message{
				Detail: model.MessageDetail{
					Actor:  "fil/3/multisig",
					Method: "Propose",
					Params: &multisig3.ProposeParams{},
				},
			},
		},
		schema.Model{
			Name: "exec-trace: multisig.ProposeReturn v3",
			D:    execTraceExample,
		},
		schema.Model{
			Name: "exec-gas",
			D:    &model.ExecGas{},
		},
		schema.Model{
			Name: "fil-supply",
			D:    &model.FilSupply{},
		},
	)
}

var extractors = []extractor{
	{
		name:   "tipset",
		method: extractTipSet,
	},
	{
		name:   "block-header",
		method: extractBlochHeaders,
	},
	{
		name:   "exec-trace",
		method: extractExecTrace,
	},
	{
		name:   "actor-head",
		method: extractActorHead,
	},
}

type extractor struct {
	name   string
	method func(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error
}

// Extract tries to take all data out of specified tipset
func Extract(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error {
	tlog := ctx.L.With("epoch", ts.Height())

	for ei := range extractors {
		start := time.Now()
		if err := extractors[ei].method(ctx, res, ts); err != nil {
			return fmt.Errorf("extracting %s: %w", extractors[ei].name, err)
		}
		tlog.Infow("tipset extractor done", "name", extractors[ei].name, "elapsed", time.Now().Sub(start).String())
	}

	return nil
}

func extractTipSet(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error {
	doc, err := model.NewTipSet(ts)
	if err != nil {
		return err

	}

	res.Docs = append(res.Docs, doc)
	return nil
}

func extractBlochHeaders(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error {
	rawBHs := ts.Blocks()
	for bi := range rawBHs {
		bh, err := model.NewBlockHeader(rawBHs[bi])
		if err != nil {
			return err
		}

		res.Docs = append(res.Docs, bh)
	}

	return nil
}

type persistExecTrace struct {
	seq    []int
	parent *common.ExecutionTraceCompact
	exec   *common.ExecutionTraceCompact
	gas    *api.MsgGasCost
}

func walkExecTrace(seq []int, exec *common.ExecutionTraceCompact, walkFn func([]int, *common.ExecutionTraceCompact, *common.ExecutionTraceCompact)) {
	for i := range exec.Subcalls {
		subcall := &exec.Subcalls[i]
		subseq := append(seq, i)
		walkFn(subseq, exec, subcall)
		walkExecTrace(subseq, subcall, walkFn)
	}
}

func copyIndexes(src []int) []int {
	dst := make([]int, len(src))
	copy(dst, src)
	return dst
}

func extractExecTrace(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error {
	if ts.Child == nil {
		return fmt.Errorf("child is required for a *LinkedTipSet@%d", ts.Height())
	}

	elog := ctx.L.With("epoch", ts.Height())

	if isExpensive(ctx.C, ctx.D, ts) {
		// TODO: extract simple invoc results here
		elog.Warn("ignore expensive epoch exec trace")
		return nil
	}

	start := time.Now()
	st, rawinvocs, err := ctx.D.ExecutionTrace(ctx.C, ts.TipSet)
	if err != nil {
		return fmt.Errorf("tipset execution: %w", err)
	}
	elapsed := time.Now().Sub(start)

	if expect := ts.State(); st != expect {
		elog.Errorw("exec state mismatched", "expect", expect, "got", st)
	}

	var invocs []common.InvocResultCompact
	if err := mir.Mirror(&invocs, rawinvocs); err != nil {
		return fmt.Errorf("mirroring exec invoc results: %w", err)
	}

	elog.Infow("get exec invocs", "st", st, "count", len(invocs), "elapsed", elapsed.String())
	etraces := make([]persistExecTrace, 0, len(invocs)*4)
	gasTraceNames := map[string]struct{}{}

	for i := range invocs {
		exec := &invocs[i].ExecutionTrace
		etraces = append(etraces, persistExecTrace{
			seq:    []int{i},
			parent: nil,
			exec:   exec,
			gas:    &invocs[i].GasCost,
		})

		for ni := range exec.GasCharges {
			name := exec.GasCharges[ni].Name
			if _, has := gasTraceNames[name]; !has {
				gasTraceNames[name] = struct{}{}
			}
		}

		walkExecTrace([]int{i}, exec, func(subseq []int, subparent, subexec *common.ExecutionTraceCompact) {
			etraces = append(etraces, persistExecTrace{
				seq:    copyIndexes(subseq),
				parent: subparent,
				exec:   subexec,
			})

			for ni := range subexec.GasCharges {
				name := subexec.GasCharges[ni].Name
				if _, has := gasTraceNames[name]; !has {
					gasTraceNames[name] = struct{}{}
				}
			}
		})
	}

	gtnames := make([]string, 0, len(gasTraceNames))
	for gn := range gasTraceNames {
		gtnames = append(gtnames, gn)
	}
	sort.Strings(gtnames)

	if err := ctx.D.AddEnum(ctx.C, model.NSGasTraceNames, gtnames...); err != nil {
		return fmt.Errorf("add gas trace names: %w", err)
	}

	dupmsgs := map[cid.Cid]struct{}{}

	var msgcnt, tracecnt int

	for i := range etraces {
		p := etraces[i]
		msg := &p.exec.Msg

		var parentMsg *types.Message
		if p.parent != nil {
			parentMsg = &p.parent.Msg
		}

		mi, err := ctx.Actors.Set.LookupMethodInfo(ctx.C, ts.TipSet, ctx.D, parentMsg, msg)
		if err != nil {
			if !errors.Is(err, actor.ErrActorMethodNotFound) {
				return fmt.Errorf("lookup method info for %s/%d: %w", msg.To, msg.Method, err)
			}

			elog.Errorf("%s", err)
		}

		mcid := msg.Cid()
		if _, has := dupmsgs[mcid]; !has {
			mmsg, err := model.NewMessage(mcid, msg, mi.Actor, mi.Method.Name, mi.ParamObj())
			if err != nil {
				elog.Errorw("convert to model.Message", "mcid", mcid, "from", msg.From, "to", msg.To, "actor", mi.Actor, "method", mi.Method.Name, "err", err.Error())
			} else {
				res.Docs = append(res.Docs, mmsg)
				msgcnt++
				dupmsgs[mcid] = struct{}{}
			}
		}

		met, meg, err := model.NewExecTrace(ctx.C, ctx.D, mcid, ts.Height(), p.seq, p.exec, mi.ReturnObj(), p.gas)
		if err != nil {
			elog.Errorw("convert to model.MessageExec", "mcid", mcid, "from", msg.From, "to", msg.To, "actor", mi.Actor, "method", mi.Method.Name, "err", err.Error())
		} else {
			tracecnt++
			res.Docs = append(res.Docs, met)
			if meg != nil && len(meg.Charges) > 0 {
				res.Docs = append(res.Docs, meg)
			}
		}

	}

	elog.Infow("converted from raw to model", "msg", msgcnt, "exec-trace", tracecnt)

	return nil
}

func extractActorHead(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet) error {
	height := ts.Height()

	forRegular := ctx.Opts.StateRegular.Interval > 0 && height%ctx.Opts.StateRegular.Interval == 0

	if !forRegular {
		return nil
	}

	root := ts.ParentState()
	tree, err := ctx.D.StateTree(root)
	if err != nil {
		return fmt.Errorf("load state tree for %s: %w", root, err)
	}

	supply, err := ctx.D.GetVMCirculatingSupplyDetailed(ctx.C, height, tree)
	if err != nil {
		return fmt.Errorf("get vm circulating supply: %w", err)
	}

	count := 0
	actors := []*common.ActorHead{}
	var powerActor *types.Actor
	err = tree.ForEach(func(addr address.Address, act *types.Actor) error {
		if addr == builtin.SystemActorAddr || addr == builtin.CronActorAddr || lbuiltin.IsAccountActor(act.Code) {
			return nil
		}

		if addr == builtin.StoragePowerActorAddr {
			powerActor = act
		}

		actors = append(actors, &common.ActorHead{
			Actor:             act,
			CirculatingSupply: &supply,
			Addr:              addr,
			Epoch:             height,
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("walk through all actors: %w", err)
	}

	elog := ctx.L.With("epoch", height)
	elog.Infow("actor heads extracted", "count", count, "valuable", len(actors))

	for ai := range actors {
		actors[ai].Global.Power = powerActor
	}

	res.RegularStates = actors

	res.Docs = append(res.Docs, &model.FilSupply{
		Epoch:             height,
		CirculatingSupply: supply,
	})

	return nil
}
