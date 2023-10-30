package tipset

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/go-address"
	amt4 "github.com/filecoin-project/go-amt-ipld/v4"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	builtintypes "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v10/evm"
	"github.com/filecoin-project/go-state-types/builtin/v11/miner"
	sverifreg "github.com/filecoin-project/go-state-types/builtin/v11/verifreg"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	builtin2 "github.com/filecoin-project/lotus/chain/actors/builtin"
	_init "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/actors/builtin/market"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/filecoin-project/lotus/chain/vm"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mir"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
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
		schema.Model{
			Name: "final-height",
			D:    &model.FinalHeight{},
		},
		//schema.Model{
		//	Name: "message-block",
		//	D:    &model.MessageBlock{},
		//},
		schema.Model{
			Name: "block-message",
			D:    &model.BlockMessage{},
		},

		schema.Model{
			Name: "actor-message",
			D:    &model.ActorMessage{},
		},
		schema.Model{
			Name: "state-final-height",
			D:    &model.StateFinalHeight{},
		},

		schema.Model{
			Name: "eth-hash",
			D:    &model.EthHash{},
		},
		schema.Model{
			Name: "events-root",
			D:    &model.EventsRoot{},
		},
		schema.Model{
			Name: "explicit-message",
			D:    &model.ExplicitMessage{},
		},
		schema.Model{
			Name: "evm-initcode",
			D:    &model.EvmInitCode{},
		},
		schema.Model{
			Name: "actor-event",
			D:    &model.ActorEvent{},
		},
		schema.Model{
			Name: "miner-sector",
			D:    &model.MinerSector{},
		},
		schema.Model{
			Name: "sector-claim",
			D:    &model.SectorClaim{},
		},
		schema.Model{
			Name: "new-deal-proposal",
			D:    &model.NewDealProposal{},
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
	{
		name:   "actor-balance",
		method: extractActorBalance,
	},
	//{
	//	name:   "message-block",
	//	method: extractMessageBlock,
	//},
	{
		name:   "block-message",
		method: extractBlockMessage,
	},
	{
		name:   "actor-address",
		method: extractActorAddress,
	},
	{
		name:   "changed-actor",
		method: extractChangedActor,
	},
	{
		name:   "new-dealproposal",
		method: extractDealProposal,
	},
}

type extractor struct {
	name   string
	method func(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, allowNilChild bool) error
}

// Extract tries to take all data out of specified tipset
func Extract(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, allowNilChild bool) error {
	tlog := ctx.L.With("epoch", ts.Height())

	for ei := range extractors {
		start := time.Now()
		if err := extractors[ei].method(ctx, res, ts, allowNilChild); err != nil {
			return fmt.Errorf("extracting %s: %w", extractors[ei].name, err)
		}

		if allowNilChild && extractors[ei].name == "tipset" {
			continue
		}
		tlog.Infow("tipset extractor done", "name", extractors[ei].name, "elapsed", time.Now().Sub(start).String())
	}

	return nil
}

func extractTipSet(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error { // nolint: deadcode
	if !ctx.Opts.EnabelExtract.EnableExtractTipset {
		return nil
	}
	if tmp {
		return nil
	}

	doc, err := model.NewTipSet(ts)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, doc)
	return nil
}

func extractTipSetForTmp(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, st cid.Cid) error { // nolint: deadcode
	weight, err := ctx.D.Weight(ctx.C, ts.TipSet)
	if err != nil {
		return fmt.Errorf("get weight of tipset failed: %v", err)
	}
	baseFee, err := ctx.D.ComputeBaseFee(ctx.C, ts.TipSet)
	if err != nil {
		return fmt.Errorf("get basefee of tipset failed: %v", err)
	}
	doc, err := model.NewTipSetWithoutChild(ts, weight, baseFee, st)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, doc)
	return nil
}

func extractBlochHeaders(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error { // nolint: deadcode
	if !ctx.Opts.EnabelExtract.EnableExtractBlockHeader {
		return nil
	}

	rawBHs := ts.Blocks()
	for bi := range rawBHs {
		minerID, err := extract.LookupID(ctx, rawBHs[bi].Miner, ts.TipSet)
		if err != nil {
			return err
		}

		bh, err := model.NewBlockHeader(minerID, rawBHs[bi])
		if err != nil {
			return err
		}

		bmsgs, smsgs, err := ctx.D.MessagesForBlock(ctx.C, rawBHs[bi])
		if err != nil {
			return err
		}

		bh.MessageCount = len(bmsgs) + len(smsgs)
		res.Docs = append(res.Docs, bh)
	}

	return nil
}

type persistExecTrace struct {
	seq    []int
	parent *common.ExecutionTraceCompact
	exec   *common.ExecutionTraceCompact
	gas    *api.MsgGasCost
	errMsg string

	// not internal msg fields
	nonce      uint64
	rootCid    cid.Cid
	gasFeeCap  abi.TokenAmount
	gasPremium abi.TokenAmount
	eventRoot  *cid.Cid
	rctVersion types.MessageReceiptVersion
}

func (p persistExecTrace) info() string {
	return fmt.Sprintf("rootcid: %s,from: %s,to: %s", p.rootCid, p.exec.Msg.From, p.parent.Msg.To)
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

func extractExecTrace(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {

	_, span := trace.StartSpan(ctx.C, "extractor.extractExecTrace")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()

	if !tmp && ts.Child == nil {
		return fmt.Errorf("child is required for a *LinkedTipSet@%d", ts.Height())
	}

	elog := ctx.L.With("epoch", ts.Height())

	if ctx.Opts.SkipExpensiveEpoch && isExpensive(ctx.C, ctx.D, ts) {
		// TODO: extract simple invoc results here
		elog.Warn("ignore expensive epoch exec trace")
		return nil
	}

	if !ctx.Opts.EnabelExtract.EnableExtractExecTrace && !ctx.Opts.EnabelExtract.EnableExtractMessage && !ctx.Opts.EnabelExtract.EnableExtractActorMessage && !ctx.Opts.EnabelExtract.EnableExtractEthHash && !ctx.Opts.EnabelExtract.EnableExtractEventsRoot &&
		!ctx.Opts.EnabelExtract.EnableExtractExplicitMessage && !ctx.Opts.EnabelExtract.EnableExtractEvmByteCode && !ctx.Opts.EnabelExtract.EnableExtractActorEvent && !ctx.Opts.EnabelExtract.EnableExtractMinerSector && !ctx.Opts.EnabelExtract.EnableExtractSectorClaim && !ctx.Opts.EnabelExtract.EnableExtractCreateMessage {
		return nil
	}

	start := time.Now()
	st, rawinvocs, err := ctx.D.ExecutionTrace(ctx.C, ts.TipSet)
	if err != nil {
		return fmt.Errorf("tipset.Height: %v, tipset.Cids: %v, pstate: %v, tipset execution failed: %w", ts.TipSet.Height().String(), ts.TipSet.Cids(), ts.TipSet.Blocks()[0].ParentStateRoot, err)
	}

	if tmp {
		elog.Infof("tipset execution successfully, tipset.Height: %v, tipset.Cids: %v, state: %v, pstate: %v", ts.TipSet.Height().String(), ts.TipSet.Cids(), st.String(), ts.TipSet.Blocks()[0].ParentStateRoot)
	}

	elapsed := time.Now().Sub(start)

	if ts.Child != nil {
		if expect := ts.State(); st != expect {
			return fmt.Errorf("exec state of tipset %v mismatched, expect: %v, got: %v", ts.TipSet.Height(), expect, st)
		}
	}

	var invocs []common.InvocResultCompact
	if err := mir.Mirror(&invocs, rawinvocs); err != nil {
		return fmt.Errorf("mirroring exec invoc results: %w", err)
	}

	elog.Infow("get exec invocs", "st", st, "count", len(invocs), "elapsed", elapsed.String())

	if tmp {
		// todo: call TipSetState to get Receipts?
		tstart := time.Now()
		err = extractTipSetForTmp(ctx, res, ts, st)
		if err != nil {
			return fmt.Errorf("extract tipset failed: %v", err)
		}
		elog.Infow("tipset extractor done", "name", "tipset", "elapsed", time.Now().Sub(tstart).String())
	}

	etraces := make([]persistExecTrace, 0, len(invocs)*4)
	callerAddrMap := make(map[string]address.Address)
	// [2]cid.Cid Cid,SingedCid
	IDCidMap := make(map[string][2]cid.Cid)
	gasTraceNames := map[string]struct{}{}

	for i := range invocs {
		exec := &invocs[i].ExecutionTrace
		errMsg := invocs[i].Error
		nonce := invocs[i].RawMsg.Nonce
		etraces = append(etraces, persistExecTrace{
			seq:        []int{i},
			parent:     nil,
			exec:       exec,
			gas:        &invocs[i].GasCost,
			errMsg:     errMsg,
			nonce:      nonce,
			rootCid:    invocs[i].MsgCid,
			gasFeeCap:  invocs[i].RawMsg.GasFeeCap,
			gasPremium: invocs[i].RawMsg.GasPremium,
			eventRoot:  invocs[i].MsgRct.EventsRoot,
			rctVersion: invocs[i].MsgRct.Version(),
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
				errMsg: errMsg,
				nonce:  nonce,
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

	allmsgs, err := ctx.D.MessagesForTipset(ctx.C, ts.TipSet)
	if err != nil {
		return fmt.Errorf("get messages for tipset err: %w", err)
	}

	allmsgsMap := make(map[string]types.ChainMsg)
	allmsgsCidMap := make(map[string]types.ChainMsg)
	for _, msg := range allmsgs {
		key := msg.VMMessage().From.String() + "-" + strconv.FormatUint(msg.VMMessage().Nonce, 10)
		if _, ok := allmsgsMap[key]; ok {
			continue
		}
		allmsgsMap[key] = msg

		if v, ok := msg.(*types.SignedMessage); ok {
			allmsgsCidMap[v.Cid().String()] = msg
		} else {
			allmsgsCidMap[msg.Cid().String()] = msg
		}

	}

	av, err := actors.VersionForNetwork(ctx.D.GetNetworkVersion(ctx.C, ts.Height()))
	if err != nil {
		return fmt.Errorf("get version for network failed: %v", err)
	}

	var msgcnt, tracecnt, actorMsgCnt, createMsgCnt, ethCnt, etcnt, emtCnt, initCodeCnt, aecnt, mstCnt, sctCnt int

	if ctx.Opts.EnabelExtract.EnableExtractEthHash {
		for _, cmsg := range allmsgs {
			smsg, ok := cmsg.(*types.SignedMessage)
			if !ok {
				continue
			}

			if smsg.Signature.Type != crypto.SigTypeDelegated {
				continue
			}

			hash, err := newEthTxFromSignedMessage(ctx.C, smsg, ts.TipSet, ctx.D)
			if err != nil {
				return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, smsg.Cid())
			}

			eht, err := model.NewEthHash(hash, smsg.Cid(), ts.Height())
			if err != nil {
				elog.Warnw("convert to model.EthHash", "mcid", smsg.Cid(), "err", err.Error())
			} else {
				ethCnt++
				res.Docs = append(res.Docs, eht)
				elog.Infow("add to ethhash", "mcid", smsg.Cid(), "hash", hash.String())
			}
		}
	}

	dupmsgs := map[cid.Cid]struct{}{}

	for i := range etraces {
		var mcid cid.Cid
		p := etraces[i]
		depth := len(p.seq)
		msg := &p.exec.Msg
		if depth == 1 {
			mcid = p.rootCid

			if v, ok := allmsgsCidMap[mcid.String()]; ok {
				msg.From = v.VMMessage().From
				msg.To = v.VMMessage().To
			}
		} else {
			mcid, err = common.MsgTraceCid(&p.exec.Msg)
			if err != nil {
				elog.Warnf("MsgTraceCid err: %w", err)
			}
		}

		var parentMsg *types.MessageTrace
		if p.parent != nil {
			parentMsg = &p.parent.Msg
		}

		mi, err := ctx.Actors.Set.LookupMethodInfo(ctx.C, ts.TipSet, ctx.D, parentMsg, msg)
		if err != nil {
			if !errors.Is(err, actor.ErrActorMethodNotFound) {
				return fmt.Errorf("lookup method info for %s/%d: %w", msg.To, msg.Method, err)
			}

			elog.Warnf("%s", err)
		}

		var signedCid cid.Cid

		key := msg.From.String() + "-" + strconv.FormatUint(p.nonce, 10)
		if cmsg, ok := allmsgsMap[key]; ok {
			smsg, ok := cmsg.(*types.SignedMessage)
			if ok {
				signedCid = mcid
				if mcid != smsg.Cid() {
					signedCid = smsg.Cid()
					elog.Infow("new messagecid", "newMcid", signedCid, "oldMcid", mcid)
				}
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractMessage {
			if _, has := dupmsgs[mcid]; !has {
				mmsg, err := model.NewMessage(mcid, signedCid, msg, mi.Actor, mi.Method.Name, mi.ParamObj(), ts.Height(), p.gasFeeCap, p.gasPremium)
				if err != nil {
					elog.Warnw("convert to model.Message", "mcid", mcid, "signedCid", signedCid, "from", msg.From, "to", msg.To, "actor", mi.Actor, "method", mi.Method.Name, "err", err.Error())
				} else {
					res.Docs = append(res.Docs, mmsg)
					msgcnt++
					dupmsgs[mcid] = struct{}{}
				}
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractExecTrace {
			isBlock := IsBlock(p.seq, msg.From)
			met, _, err := model.NewExecTrace(ctx, mcid, signedCid, ts.Height(), p.seq, p.exec, mi.ReturnObj(), p.gas, mi.Method.Name, isBlock, IDCidMap)
			if err != nil {
				elog.Warnw("convert to model.MessageExec", "mcid", mcid, "signedCid", signedCid, "from", msg.From, "to", msg.To, "actor", mi.Actor, "method", mi.Method.Name, "err", err.Error())
			} else {
				tracecnt++
				// update callerAddrMap
				callerAddrMap[met.ID] = met.Msg.From
				if met.IsBlock {
					IDCidMap[met.ID] = [2]cid.Cid{met.Cid, met.SignedCid}
				}

				elog.Debug(IDCidMap)
				res.Docs = append(res.Docs, met)
				//if meg != nil && len(meg.Charges) > 0 {
				//	res.Docs = append(res.Docs, meg)
				//}
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractEventsRoot || ctx.Opts.EnabelExtract.EnableExtractActorEvent {
			if p.eventRoot != nil && p.rctVersion == types.MessageReceiptV1 {
				eventsRoot := p.eventRoot
				if eventsRoot != nil {
					events, err := GetEvents(ctx.C, *eventsRoot, ctx.D)
					if err != nil {
						return fmt.Errorf("get events failed: %v, eventsRoot: %v, mcid: %v, signedCid: %v", err, eventsRoot, mcid, signedCid)
					}

					if ctx.Opts.EnabelExtract.EnableExtractEventsRoot {
						etm, err := model.NewEventsRoot(*eventsRoot, events, ts.Height())
						if err != nil {
							elog.Warnw("convert to model.EventsRoot", "eventsRoot", eventsRoot, "mcid", mcid, "signedCid", signedCid, "err", err.Error())
						} else {
							res.Docs = append(res.Docs, etm)
							etcnt++
						}
					}

					if ctx.Opts.EnabelExtract.EnableExtractActorEvent {
						for i, evt := range events {
							actorID, err := address.NewIDAddress(uint64(evt.Emitter))
							if err != nil {
								return fmt.Errorf("failed to create ID address: %w", err)
							}

							data, topics, ok := ethLogFromEvent(ctx, ts.TipSet, evt.Entries)
							if !ok {
								// not an eth event.
								elog.Warnw("ethLogFromEvent not an eth event", "actorID", actorID, "mcid", mcid, "signedCid", signedCid)
								//continue //todo
							}

							logIndex := uint64(i)
							removed := false
							aet, err := model.NewActorEvent(actorID, ts.Height(), mcid, signedCid, topics, data, logIndex, removed, p.seq)
							if err != nil {
								elog.Warnw("convert to model.ActorEvent", "actorID", actorID, "mcid", mcid, "signedCid", signedCid, "err", err.Error())
							} else {
								aecnt++
								res.Docs = append(res.Docs, aet)
							}
						}
					}
				}
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractActorMessage {
			isBlock := IsBlock(p.seq, msg.From)
			storeMap := make(map[address.Address]string)

			fromActorID, err := extract.LookupID(ctx, msg.From, ts.TipSet)
			if err != nil {
				elog.Warnf("lookup ID for %v at %v failed: %v", msg.From, ts.Height(), err)
				fromActorID = msg.From
			}

			if _, ok := storeMap[fromActorID]; !ok {
				storeMap[fromActorID] = "from"
			}

			toActorID, err := extract.LookupID(ctx, msg.To, ts.TipSet)
			if err != nil {
				elog.Warnf("lookup ID for %v at %v failed: %v", msg.To, ts.Height(), err)
				toActorID = msg.To
			}

			if _, ok := storeMap[toActorID]; !ok {
				storeMap[toActorID] = "to"
			}

			for ID, mtype := range storeMap {
				transferType := GetTransferType(msg.From, msg.To, mtype, mi.Method.Name, msg.Value)
				amsg, err := model.NewActorMessage(ctx, ID, ts.Height(), mcid, signedCid, msg.Value, mi.Method.Name, p.exec.MsgRct.ExitCode, mtype, msg.From, msg.To, isBlock, p.seq, transferType, IDCidMap)
				if err != nil {
					elog.Warnw("convert to model.ActorMessage", "actorID", ID, "mcid", mcid, "signedCid", signedCid)
				} else {
					actorMsgCnt++
					res.Docs = append(res.Docs, amsg)
				}
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractCreateMessage {
			isBlock := IsBlock(p.seq, msg.From)
			method := mi.Method.Name
			if model.IsOkCreateMessage(method, int64(p.exec.MsgRct.ExitCode)) {
				cmsg, err := model.NewCreateMessage(ctx, ts.Height(), mcid, signedCid, msg.Value, mi.Method.Name, msg.From, msg.To, isBlock, p.seq, callerAddrMap, mi.ReturnObj(), p.exec, IDCidMap)

				if err != nil {
					elog.Warnw("convert to model.CreateMessage", "mcid", mcid, "signedCid", signedCid)
				} else {
					createMsgCnt++
					res.Docs = append(res.Docs, cmsg)
				}
			}

		}

		if ctx.Opts.EnabelExtract.EnableExtractExplicitMessage {
			if cmsg, ok := allmsgsMap[key]; ok {
				var exitCode exitcode.ExitCode
				if p.exec != nil {
					exitCode = p.exec.MsgRct.ExitCode
				}

				emt := model.NewExplicitMessage(cmsg.Cid(), ts.Height(), msg.Value, mi.Method.Name, exitCode, msg.From, msg.To)
				emtCnt++
				res.Docs = append(res.Docs, emt)
			}
		}

		if ctx.Opts.EnabelExtract.EnableExtractEvmByteCode {
			if p.exec != nil && p.exec.Msg.Method == builtintypes.MethodsEVM.Constructor && strings.Contains(mi.Actor, "evm") && p.exec.MsgRct.ExitCode.IsSuccess() {
				var params evm.ConstructorParams
				param := bytes.NewReader(p.exec.Msg.Params)
				if err := params.UnmarshalCBOR(param); err != nil {
					return fmt.Errorf("UnmarshalCBOR return value failed: %w, msg: %s", err, p.info())
				}

				eit := model.NewEvmInitCode(p.exec.Msg.To, hex.EncodeToString(params.Initcode), ts.Height())
				initCodeCnt++
				res.Docs = append(res.Docs, eit)
			}
		}

		// todo: ts.Child调用可能会出问题： 当actor在msg后销毁
		if !tmp && (ctx.Opts.EnabelExtract.EnableExtractMinerSector || ctx.Opts.EnabelExtract.EnableExtractSectorClaim) {
			var (
				minerID address.Address
				cmas    lminer.State
				err     error
			)

			if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && IsBlock(p.seq, msg.From) {
				minerID, err = extract.LookupID(ctx, msg.To, ts.TipSet)
				if err != nil {
					return fmt.Errorf("lookup ID for %v failed: %v", msg.To, err)
				}

				cmact, err := ctx.D.LoadActor(ctx.C, msg.To, ts.Child)
				if err != nil {
					return fmt.Errorf("load actor %v at %v failed: %v", msg.To, ts.Child, err)
				}

				cmas, err = lminer.Load(ctx.D.ActorStore(ctx.C), cmact)
				if err != nil {
					return fmt.Errorf("load state for miner %v failed: %v", msg.To, err)
				}

			}

			cvact, err := ctx.D.LoadActor(ctx.C, builtin.VerifiedRegistryActorAddr, ts.Child)
			if err != nil {
				return fmt.Errorf("failed to load verifiedRegistry actor at %v: %v", ts.Child.Height(), err)
			}

			cvas, err := verifreg.Load(ctx.D.ActorStore(ctx.C), cvact)
			if err != nil {
				return fmt.Errorf("failed to load verifiedRegistry state: %v", err)
			}

			cmkact, err := ctx.D.LoadActor(ctx.C, builtin.StorageMarketActorAddr, ts.Child)
			if err != nil {
				return fmt.Errorf("failed to load StorageMarketActorAddr actor at %v: %v", ts.Child.Height(), err)
			}

			cmkas, err := market.Load(ctx.D.ActorStore(ctx.C), cmkact)
			if err != nil {
				return fmt.Errorf("failed to load storageMarket state: %v", err)
			}

			// ProveCommitSector & ProveCommitAggregate generate部分sector
			if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ProveCommitSector || p.exec.Msg.Method == builtintypes.MethodsMiner.ProveCommitAggregate) {
				var sectorNumbers = bitfield.New()
				// 解析参数，version
				switch p.exec.Msg.Method {
				case builtintypes.MethodsMiner.ProveCommitSector:
					var params miner.ProveCommitSectorParams // todo
					if err := params.UnmarshalCBOR(bytes.NewReader(p.exec.Msg.Params)); err != nil {
						return fmt.Errorf("unmarshal ProveCommitSectorParams for %s failed: %v", p.info(), err)
					}

					sectorNumbers.Set(uint64(params.SectorNumber))
				case builtintypes.MethodsMiner.ProveCommitAggregate:
					var params miner.ProveCommitAggregateParams
					if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
						return fmt.Errorf("unmarshal ProveCommitAggregateParams for %s failed: %v", p.info(), err)
					}

					sectorNumbers = params.SectorNumbers
				default:
					return fmt.Errorf("invalid method: %v", p.exec.Msg.Method)
				}

				if ctx.Opts.EnabelExtract.EnableExtractMinerSector {
					// get sectors for sectorNumbers from state
					var sectorInfos []*lminer.SectorOnChainInfo
					if err := sectorNumbers.ForEach(func(i uint64) error {
						sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
						if err != nil {
							// err is nil when not found
							return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
						}

						if sectorInfo != nil {
							sectorInfos = append(sectorInfos, sectorInfo)
						}

						return nil
					}); err != nil {
						return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
					}

					for _, info := range sectorInfos {
						mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
						mstCnt++
						res.Docs = append(res.Docs, mst)
					}
				}

				if ctx.Opts.EnabelExtract.EnableExtractSectorClaim && av > actors.Version8 {
					// get claims for sectors
					claims, err := cvas.GetClaims(minerID)
					if err != nil {
						return fmt.Errorf("failed to get claims for provider: %v: %v", minerID, err)
					}

					if err := sectorNumbers.ForEach(func(i uint64) error {
						for claimID, claim := range claims {
							if claim.Sector == abi.SectorNumber(i) {
								sct := model.NewSectorClaim(uint64(claimID), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
								sctCnt++
								res.Docs = append(res.Docs, sct)
								continue
							}
						}

						return nil
					}); err != nil {
						return fmt.Errorf("load sectorclaim info for %v at %v failed: %v", msg.To, ts.Height(), err)
					}
				}
			}

			// ExtendSectorExpiration || ExtendSectorExpiration2 续期
			if ctx.Opts.EnabelExtract.EnableExtractMinerSector {
				if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration || p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration2) {
					var extendSectors []bitfield.BitField
					if p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration {
						var params miner.ExtendSectorExpirationParams
						if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
							return fmt.Errorf("unmarshal ExtendSectorExpirationParams for %s failed: %v", p.info(), err)
						}

						for _, extension := range params.Extensions {
							extendSectors = append(extendSectors, extension.Sectors)
						}
					} else if p.exec.Msg.Method == builtintypes.MethodsMiner.ExtendSectorExpiration2 {
						var params miner.ExtendSectorExpiration2Params
						if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
							return fmt.Errorf("unmarshal ExtendSectorExpiration2Params for %s failed: %v", p.info(), err)
						}

						for _, extension := range params.Extensions {
							extendSectors = append(extendSectors, extension.Sectors)
						}
					}

					for i := range extendSectors {
						sectorInfos, err := cmas.LoadSectors(&extendSectors[i])
						if err != nil {
							return fmt.Errorf("load sector infos for %v failed: %v", msg.To, err)
						}

						for _, info := range sectorInfos {
							mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
							mstCnt++
							res.Docs = append(res.Docs, mst)
						}
					}
				}
			}

			// ProveReplicaUpdates: replace proven sector
			if p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && (p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates || p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates2) {
				var (
					Deals            []abi.DealID
					succeededSectors = bitfield.New()
				)

				if p.exec.Msg.Method == builtintypes.MethodsMiner.ProveReplicaUpdates {
					var params miner.ProveReplicaUpdatesParams

					if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
						return fmt.Errorf("unmarshal ProveReplicaUpdatesParams for %s failed: %v", p.info(), err)
					}

					if err := succeededSectors.UnmarshalCBOR(bytes.NewReader(p.exec.MsgRct.Return)); err != nil {
						return fmt.Errorf("unmarshal ProveReplicaUpdates returns for %s failed: %v", p.info(), err)
					}
				} else {
					var params miner.ProveReplicaUpdatesParams2
					if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
						return fmt.Errorf("unmarshal ProveReplicaUpdatesParams2 for %s failed: %v", p.info(), err)
					}

					if err := succeededSectors.UnmarshalCBOR(bytes.NewReader(p.exec.MsgRct.Return)); err != nil {
						return fmt.Errorf("unmarshal ProveReplicaUpdates2 returns for %s failed: %v", p.info(), err)
					}
				}

				if ctx.Opts.EnabelExtract.EnableExtractMinerSector {
					var sectorInfos []*lminer.SectorOnChainInfo
					if err := succeededSectors.ForEach(func(i uint64) error {
						sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
						if err != nil {
							return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
						}

						if sectorInfo != nil {
							sectorInfos = append(sectorInfos, sectorInfo)
						}

						return nil
					}); err != nil {
						return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
					}

					for _, info := range sectorInfos {
						mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, false, ts.Height())
						mstCnt++
						res.Docs = append(res.Docs, mst)

						Deals = append(Deals, info.DealIDs...)
					}
				}

				if ctx.Opts.EnabelExtract.EnableExtractSectorClaim && av > actors.Version8 {
					var allocIDs []verifreg.AllocationId

					state, err := cmkas.States()
					if err != nil {
						return fmt.Errorf("get market state failed: %v", err)
					}

					for _, deal := range Deals {
						dealState, found, err := state.Get(deal)
						if err != nil {
							return fmt.Errorf("get dealstate failed: %v", err)
						}

						if found {
							allocIDs = append(allocIDs, dealState.VerifiedClaim)
						}
					}

					for _, allocID := range allocIDs {
						claim, found, err := cvas.GetClaim(minerID, verifreg.ClaimId(allocID))
						if err != nil {
							return fmt.Errorf("get cliam for %v of %v failed: %v", allocID, minerID, err)
						}

						if found {
							sct := model.NewSectorClaim(uint64(allocID), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
							sctCnt++
							res.Docs = append(res.Docs, sct)
						}
					}
				}
			}

			// TerminateSectors: on-time terminate or early terminate
			if ctx.Opts.EnabelExtract.EnableExtractMinerSector && p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "miner") && p.exec.Msg.Method == builtintypes.MethodsMiner.TerminateSectors {
				var params miner.TerminateSectorsParams
				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
					return fmt.Errorf("unmarshal TerminateSectorsParams for %s failed: %v", p.info(), err)
				}

				var terminateSectors = bitfield.New()
				for _, termination := range params.Terminations {
					var err error
					if terminateSectors, err = bitfield.MergeBitFields(terminateSectors, termination.Sectors); err != nil {
						return fmt.Errorf("merge termination sectors failed for %v: %v", msg.To, err)
					}
				}

				var sectorInfos []*lminer.SectorOnChainInfo
				if err := terminateSectors.ForEach(func(i uint64) error {
					sectorInfo, err := cmas.GetSector(abi.SectorNumber(i))
					if err != nil {
						// err is nil when not found
						return fmt.Errorf("get sector %v for %v failed: %v", i, msg.To, err)
					}

					if sectorInfo != nil {
						sectorInfos = append(sectorInfos, sectorInfo)
					}

					return nil
				}); err != nil {
					return fmt.Errorf("load sector info for %v at %v failed: %v", msg.To, ts.Height(), err)
				}

				for _, info := range sectorInfos {
					mst := model.NewMinerSector(minerID, info.SectorNumber, info.DealIDs, info.Activation, info.Expiration, info.DealWeight, info.VerifiedDealWeight, info.SimpleQAPower, info.InitialPledge, true, ts.Height())
					mstCnt++
					res.Docs = append(res.Docs, mst)
				}
			}

			// ExtendClaimTerms allows Partial failure
			if ctx.Opts.EnabelExtract.EnableExtractSectorClaim && p.exec != nil && p.exec.MsgRct.ExitCode.IsSuccess() && strings.Contains(mi.Actor, "verifiedregistry") && (p.exec.Msg.Method == builtintypes.MethodsVerifiedRegistry.ExtendClaimTerms || p.exec.Msg.Method == builtintypes.MethodsVerifiedRegistry.ExtendClaimTermsExported) {
				var params sverifreg.ExtendClaimTermsParams
				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
					return fmt.Errorf("unmarshal ExtendClaimTermsParams for %s failed: %v", p.info(), err)
				}

				for _, term := range params.Terms {
					providerID, err := address.NewIDAddress(uint64(term.Provider))
					if err != nil {
						return fmt.Errorf("new id address for %v failed: %v", term.Provider, err)
					}

					claim, found, err := cvas.GetClaim(providerID, verifreg.ClaimId(term.ClaimId))
					if err != nil || !found {
						return fmt.Errorf("get claim for %v of %v failed: %v", providerID, term.ClaimId, err)
					}

					sct := model.NewSectorClaim(uint64(term.ClaimId), claim.Provider, claim.Client, claim.Data, claim.Size, claim.TermMin, claim.TermMax, claim.TermStart, claim.Sector, ts.Height())
					sctCnt++
					res.Docs = append(res.Docs, sct)
				}
			}
		}
	}

	elog.Infow("converted from raw to model", "msg", msgcnt, "exec-trace", tracecnt, "actor-message", actorMsgCnt, "create-message", createMsgCnt, "eth-hash", ethCnt, "events-root", etcnt, "explicit-message", emtCnt, "evm-initcode", initCodeCnt, "actor-event", aecnt, "miner-sector", mstCnt, "sector-claim", sctCnt)

	return nil
}

func IsBlock(seq []int, from address.Address) bool {
	return len(seq) == 1 && (strings.HasPrefix(from.String()[1:], "1") || strings.HasPrefix(from.String()[1:], "3") || strings.HasPrefix(from.String()[1:], "4"))
}

func GetEvents(ctx context.Context, root cid.Cid, cs common.ChainStore) ([]types.Event, error) {
	store := cs.ActorStore(ctx)
	evtArr, err := amt4.LoadAMT(ctx, store, root, amt4.UseTreeBitWidth(types.EventAMTBitwidth))
	if err != nil {
		return nil, xerrors.Errorf("load events amt: %w", err)
	}

	ret := make([]types.Event, 0, evtArr.Len())
	var evt types.Event
	err = evtArr.ForEach(ctx, func(u uint64, deferred *cbg.Deferred) error {
		if u > math.MaxInt {
			return xerrors.Errorf("too many events")
		}
		if err := evt.UnmarshalCBOR(bytes.NewReader(deferred.Raw)); err != nil {
			return err
		}

		ret = append(ret, evt)
		return nil
	})

	return ret, err
}

func extractActorBalance(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	if tmp {
		return nil
	}

	_, span := trace.StartSpan(ctx.C, "extractor.extractActorBalance")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()
	height := ts.Height()
	if !common.IsZeroHour(ctx.Opts.ZeroHourExtract.ActorBalance, height) && !extract.IsExtract(ctx.Opts.StateRegular.ActorBalanceTicks, ctx, height) || !ctx.Opts.EnabelExtract.EnableExtractActorBalance {
		return nil
	}

	root := ts.ParentState()
	tree, err := ctx.D.StateTree(root)
	if err != nil {
		return fmt.Errorf("load state tree for %s: %w", root, err)
	}
	actorBalance := []*model.ActorBalance{}
	iact, err := tree.GetActor(_init.Address)
	if err != nil {
		return fmt.Errorf("failed to load init actor: %w", err)
	}
	store := ctx.D.ActorStore(ctx.C)
	ist, err := _init.Load(store, iact)
	if err != nil {
		return fmt.Errorf("failed to load init actor state: %w", err)
	}
	robustMap := make(map[address.Address]address.Address)
	err = ist.ForEachActor(func(id abi.ActorID, addr address.Address) error {
		idAddr, err := address.NewIDAddress(uint64(id))
		if err != nil {
			return fmt.Errorf("failed to write to addr map: %w", err)
		}

		robustMap[idAddr] = addr

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk through actors: %w", err)
	}

	elog := ctx.L.With("epoch", height)
	elog.Infow("actor balanced extracted")
	err = tree.ForEach(func(addr address.Address, act *types.Actor) error {
		addresses := []address.Address{addr, robustMap[addr]}
		if builtin2.IsAccountActor(act.Code) {
			pubAddr, err := vm.ResolveToDeterministicAddr(tree, store, addr)
			if err != nil {
				return err
			}
			addresses = append(addresses, pubAddr)
		}
		id, err := actorstate.GenRegularHeadID(act.Head, addr, height)
		if err != nil {
			return fmt.Errorf("generate regular id: %w", err)
		}
		actType := builtin2.ActorNameByCode(act.Code)
		actTypes := strings.Split(actType, "/")
		if len(actTypes) > 1 {
			actType = actTypes[len(actTypes)-1]
		} else {
			elog.Warnf("actor %s acttype out of design", actType)
		}

		actorBalance = append(actorBalance, &model.ActorBalance{
			ActorStateExBasic: model.ActorStateExBasic{
				ID:    id,
				Path:  []cid.Cid{act.Head},
				Addr:  addr,
				Epoch: height,
			},
			Addresses: addresses,
			Balance:   act.Balance,
			Code:      actType,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk through all actors: %w", err)
	}

	for i := range actorBalance {
		res.Docs = append(res.Docs, actorBalance[i])
	}

	return nil
}

func extractActorHead(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	if tmp {
		return nil
	}

	_, span := trace.StartSpan(ctx.C, "extractor.extractActorHead")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()
	height := ts.Height()

	forRegular := ctx.Opts.StateRegular.Interval > 0 && height%ctx.Opts.StateRegular.Interval == 0

	var extractEvenNullTipSet bool
	for h := ts.Parent.Height() + 1; h <= ts.Height(); h++ {
		if ctx.Opts.StateRegular.Interval > 0 && h%ctx.Opts.StateRegular.Interval == 0 || common.IsZeroHour(true, h) {
			extractEvenNullTipSet = true
			height = h
			break
		}
	}

	if !extractEvenNullTipSet || !ctx.Opts.EnabelExtract.EnableExtractState && !ctx.Opts.EnabelExtract.EnableExtractFilSupply {
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

	if ctx.Opts.EnabelExtract.EnableExtractState {
		count := 0
		actors := []*common.ActorHead{}
		var powerActor *types.Actor
		err = tree.ForEach(func(addr address.Address, act *types.Actor) error {
			count++
			if addr == builtin.SystemActorAddr || addr == builtin.CronActorAddr {
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
				TipSet:            ts.TipSet,
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
	}

	if ctx.Opts.EnabelExtract.EnableExtractFilSupply && (forRegular || common.IsZeroHour(ctx.Opts.ZeroHourExtract.FilSupply, height)) {
		res.Docs = append(res.Docs, &model.FilSupply{
			Epoch:             height,
			CirculatingSupply: supply,
		})
	}

	return nil
}

//func extractMessageBlock(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
//	if !ctx.Opts.EnabelExtract.EnableExtractMessageBlock {
//		return nil
//	}
//
//	_, span := trace.StartSpan(ctx.C, "extractor.extractMessageBlock")
//	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
//	defer span.End()
//
//	elog := ctx.L.With("epoch", ts.Height())
//	elog.Infow("message-block extracted")
//
//	messageBlockMap := make(map[cid.Cid][]cid.Cid)
//	for _, blk := range ts.Blocks() {
//		// contain messages before the replacement
//		bms, sms, err := ctx.D.MessagesForBlock(ctx.C, blk)
//		if err != nil {
//			return fmt.Errorf("get messages for block %v failed: %v", blk.Cid().String(), err)
//		}
//
//		blockMessages := make([]types.ChainMsg, 0)
//		for _, bmsg := range bms {
//			blockMessages = append(blockMessages, bmsg)
//		}
//		for _, smsg := range sms {
//			blockMessages = append(blockMessages, smsg)
//		}
//
//		for _, message := range blockMessages {
//			if _, ok := messageBlockMap[message.Cid()]; !ok {
//				messageBlockMap[message.Cid()] = []cid.Cid{blk.Cid()}
//			} else {
//				messageBlockMap[message.Cid()] = append(messageBlockMap[message.Cid()], blk.Cid())
//			}
//		}
//	}
//
//	for mcid, bcids := range messageBlockMap {
//		messageBlock, _ := model.NewMessageBlock(mcid, ts.Height(), bcids)
//		res.Docs = append(res.Docs, messageBlock)
//	}
//
//	return nil
//}

func extractBlockMessage(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	if !ctx.Opts.EnabelExtract.EnableExtractBlockMessage {
		return nil
	}

	_, span := trace.StartSpan(ctx.C, "extractor.extractBlockMessage")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()

	elog := ctx.L.With("epoch", ts.Height())
	elog.Infow("block-message extracted")

	blockMessageMap := make(map[cid.Cid][]cid.Cid)
	for _, blk := range ts.Blocks() {
		// contain messages before the replacement
		bms, sms, err := ctx.D.MessagesForBlock(ctx.C, blk)
		if err != nil {
			return fmt.Errorf("get messages for block %v failed: %v", blk.Cid().String(), err)
		}

		for _, bmsg := range bms {
			blockMessageMap[blk.Cid()] = append(blockMessageMap[blk.Cid()], bmsg.Cid())
		}
		for _, smsg := range sms {
			blockMessageMap[blk.Cid()] = append(blockMessageMap[blk.Cid()], smsg.Cid())
		}
	}

	for bcid, mcids := range blockMessageMap {
		blockMessage, _ := model.NewBlockMessage(bcid, ts.Height(), mcids)
		res.Docs = append(res.Docs, blockMessage)
	}

	return nil
}

func extractActorAddress(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	if tmp {
		return nil
	}

	_, span := trace.StartSpan(ctx.C, "extractor.extractActorAddress")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()
	height := ts.Height()
	if !common.IsZeroHour(ctx.Opts.ZeroHourExtract.ActorAddress, height) && !extract.IsExtract(ctx.Opts.StateRegular.ActorAddressTicks, ctx, height) || !ctx.Opts.EnabelExtract.EnableExtractActorAddress {
		return nil
	}

	// todo: 对整点的null tipset不强求
	root := ts.ParentState()
	tree, err := ctx.D.StateTree(root)
	if err != nil {
		return fmt.Errorf("load state tree for %s: %w", root, err)
	}
	actorAddresses := []*model.ActorAddress{}
	iact, err := tree.GetActor(_init.Address)
	if err != nil {
		return fmt.Errorf("failed to load init actor: %w", err)
	}
	store := ctx.D.ActorStore(ctx.C)
	ist, err := _init.Load(store, iact)
	if err != nil {
		return fmt.Errorf("failed to load init actor state: %w", err)
	}
	robustMap := make(map[address.Address][]address.Address)
	err = ist.ForEachActor(func(id abi.ActorID, addr address.Address) error {
		idAddr, err := address.NewIDAddress(uint64(id))
		if err != nil {
			return fmt.Errorf("failed to write to addr map: %w", err)
		}

		robustMap[idAddr] = append(robustMap[idAddr], addr)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk through actors: %w", err)
	}

	elog := ctx.L.With("epoch", height)
	elog.Infow("actor address extracted")
	for actorID, addresses := range robustMap {
		actorAddress := &model.ActorAddress{ActorID: actorID, Epoch: ts.Height()}
		for _, addr := range addresses {
			switch addr.Protocol() {
			case address.SECP256K1, address.Actor, address.BLS:
				actorAddress.RobustAddress = addr
			case address.Delegated:
				actorAddress.DelegatedAddress = addr
			default:
				return fmt.Errorf("invalid address for %v, addr: %v, protocol: %v", actorID, addr, addr.Protocol())
			}
		}

		actorAddresses = append(actorAddresses, actorAddress)
	}

	for i := range actorAddresses {
		res.Docs = append(res.Docs, actorAddresses[i])
	}

	return nil
}

func extractChangedActor(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	_, span := trace.StartSpan(ctx.C, "extractor.extractChangedActor")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()
	height := ts.Height()
	if !ctx.Opts.EnabelExtract.EnableExtractChangedActor {
		return nil
	}

	elog := ctx.L.With("epoch", height)
	elog.Infow("changed actor extracted")

	old := ts.Parent.ParentState()
	new := ts.ParentState()

	oldTree, err := ctx.D.StateTree(old)
	if err != nil {
		return fmt.Errorf("failed to load old state tree: %w", err)
	}

	newTree, err := ctx.D.StateTree(new)
	if err != nil {
		return fmt.Errorf("failed to load new state tree: %w", err)
	}

	changedActors, err := state.Diff(ctx.C, oldTree, newTree)
	if err != nil {
		return err
	}

	changedActor := []*model.ChangedActor{}
	for addr, act := range changedActors {
		actorID, err := address.NewFromString(addr)
		if err != nil {
			return fmt.Errorf("new address for addr: %v faile: %v", addr, err)
		}
		id, err := actorstate.GenRegularHeadID(act.Head, actorID, height)
		if err != nil {
			return fmt.Errorf("generate regular id: %w", err)
		}
		actType := builtin2.ActorNameByCode(act.Code)
		actTypes := strings.Split(actType, "/")
		if len(actTypes) > 1 {
			actType = actTypes[len(actTypes)-1]
		} else {
			elog.Warnf("actor %s acttype out of design", actType)
		}
		changedActor = append(changedActor, &model.ChangedActor{
			ID:      id,
			Epoch:   height,
			ActorID: actorID,
			Balance: act.Balance,
			Code:    actType,
			Address: act.Address,
		})
	}

	for i := range changedActor {
		res.Docs = append(res.Docs, changedActor[i])
	}

	return nil
}

func newEthTxFromSignedMessage(ctx context.Context, smsg *types.SignedMessage, ts *types.TipSet, sm common.StateManager) (ethtypes.EthHash, error) {
	var tx ethtypes.EthTx
	var err error

	if smsg.Signature.Type == crypto.SigTypeDelegated {
		tx, err = ethtypes.EthTxFromSignedEthMessage(smsg)
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to convert from signed message: %w", err)
		}

		tx.Hash, err = tx.TxHash()
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to calculate hash for ethTx: %w", err)
		}

		fromAddr, err := lookupEthAddress(ctx, smsg.Message.From, ts, sm)
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
		}

		tx.From = fromAddr
	} else if smsg.Signature.Type == crypto.SigTypeSecp256k1 { // Secp Filecoin Message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), ts, sm)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	} else { // BLS Filecoin message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), ts, sm)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Message.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	}

	return tx.Hash, nil
}

func lookupEthAddress(ctx context.Context, addr address.Address, ts *types.TipSet, sm common.StateManager) (ethtypes.EthAddress, error) {
	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err == nil && !ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	if actor, err := sm.LoadActor(ctx, addr, ts); err != nil {
		return ethtypes.EthAddress{}, err
	} else if actor.Address != nil {
		if ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(*actor.Address); err == nil && !ethAddr.IsMaskedID() {
			return ethAddr, nil
		}
	}

	if err == nil && ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	idAddr, err := sm.LookupID(ctx, addr, ts)
	if err != nil {
		return ethtypes.EthAddress{}, err
	}
	return ethtypes.EthAddressFromFilecoinAddress(idAddr)
}

func ethTxFromNativeMessage(ctx context.Context, msg *types.Message, ts *types.TipSet, sm common.StateManager) ethtypes.EthTx {
	// We don't care if we error here, conversion is best effort for non-eth transactions
	from, _ := lookupEthAddress(ctx, msg.From, ts, sm)
	to, _ := lookupEthAddress(ctx, msg.To, ts, sm)
	return ethtypes.EthTx{
		To:                   &to,
		From:                 from,
		Nonce:                ethtypes.EthUint64(msg.Nonce),
		ChainID:              ethtypes.EthUint64(build.Eip155ChainId),
		Value:                ethtypes.EthBigInt(msg.Value),
		Type:                 ethtypes.Eip1559TxType,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         ethtypes.EthBigInt(msg.GasFeeCap),
		MaxPriorityFeePerGas: ethtypes.EthBigInt(msg.GasPremium),
		AccessList:           []ethtypes.EthHash{},
	}
}

func ethLogFromEvent(ctx *extract.Ctx, ts *types.TipSet, entries []types.EventEntry) (data []byte, topics []ethtypes.EthHash, ok bool) {
	elog := ctx.L.With("epoch", ts.Height())

	var (
		topicsFound      [4]bool
		topicsFoundCount int
		dataFound        bool
	)
	for _, entry := range entries {
		// Drop events with non-raw topics to avoid mistakes.
		if entry.Codec != cid.Raw {
			elog.Warnw("did not expect an event entry with a non-raw codec", "codec", entry.Codec, "key", entry.Key)
			return nil, nil, false
		}
		// Check if the key is t1..t4
		if len(entry.Key) == 2 && "t1" <= entry.Key && entry.Key <= "t4" {
			// '1' - '1' == 0, etc.
			idx := int(entry.Key[1] - '1')

			// Drop events with mis-sized topics.
			if len(entry.Value) != 32 {
				elog.Warnw("got an EVM event topic with an invalid size", "key", entry.Key, "size", len(entry.Value))
				return nil, nil, false
			}

			// Drop events with duplicate topics.
			if topicsFound[idx] {
				elog.Warnw("got a duplicate EVM event topic", "key", entry.Key)
				return nil, nil, false
			}
			topicsFound[idx] = true
			topicsFoundCount++

			// Extend the topics array
			for len(topics) <= idx {
				topics = append(topics, ethtypes.EthHash{})
			}
			copy(topics[idx][:], entry.Value)
		} else if entry.Key == "d" {
			// Drop events with duplicate data fields.
			if dataFound {
				elog.Warnw("got duplicate EVM event data")
				return nil, nil, false
			}

			dataFound = true
			data = entry.Value
		} else {
			// Skip entries we don't understand (makes it easier to extend things).
			// But we warn for now because we don't expect them.
			elog.Warnw("unexpected event entry", "key", entry.Key)
		}

	}

	// Drop events with skipped topics.
	if len(topics) != topicsFoundCount {
		elog.Warnw("EVM event topic length mismatch", "expected", len(topics), "actual", topicsFoundCount)
		return nil, nil, false
	}
	return data, topics, true
}

func GetTransferType(from, to address.Address, mtype, methodName string, value abi.TokenAmount) string {
	if mtype == "to" && from == builtintypes.RewardActorAddr && methodName == "ApplyRewards" {
		return model.Blockreward
	}

	if mtype == "from" && to == builtintypes.BurntFundsActorAddr {
		return model.Burn
	}

	if mtype == "from" && value.GreaterThan(abi.NewTokenAmount(0)) && to != builtintypes.BurntFundsActorAddr {
		return model.Send
	}

	if mtype == "to" && value.GreaterThan(abi.NewTokenAmount(0)) && methodName != "ApplyRewards" {
		return model.Receive
	}

	return model.Null
}

func extractDealProposal(ctx *extract.Ctx, res *extract.Res, ts *common.LinkedTipSet, tmp bool) error {
	if tmp || !ctx.Opts.EnabelExtract.EnableExtractNewDealProposal {
		return nil
	}

	height := ts.Height()
	elog := ctx.L.With("epoch", height)
	elog.Infow("new dealproposal extracted")

	// upgrade skip
	if ctx.Opts.SkipExpensiveEpoch && isExpensive(ctx.C, ctx.D, ts) {
		elog.Warn("ignore expensive epoch for new dealproposal")
		return nil
	}

	_, span := trace.StartSpan(ctx.C, "extractor.extractDealProposal")
	span.AddAttributes(trace.Int64Attribute("epoch", int64(ts.Height())))
	defer span.End()

	astore := ctx.D.ActorStore(ctx.C)

	av, err := actors.VersionForNetwork(ctx.D.GetNetworkVersion(ctx.C, ts.Height()))
	if err != nil {
		return fmt.Errorf("get version for network failed: %v", err)
	}

	pmact, err := ctx.D.LoadActor(ctx.C, builtin.StorageMarketActorAddr, ts.TipSet)
	if err != nil {
		return fmt.Errorf("load actor for addr: %v at height: %v failed: %v", builtin.StorageMarketActorAddr, ts.TipSet.Height(), err)
	}

	mact, err := ctx.D.LoadActor(ctx.C, builtin.StorageMarketActorAddr, ts.Child)
	if err != nil {
		return fmt.Errorf("load actor for addr: %v at height: %v failed: %v", builtin.StorageMarketActorAddr, ts.Child.Height(), err)
	}

	pmas, err := market.Load(astore, pmact)
	if err != nil {
		return fmt.Errorf("load market parent state for addr: %v failed: %v", builtin.StorageMarketActorAddr, err)
	}

	mas, err := market.Load(astore, mact)
	if err != nil {
		return fmt.Errorf("load market state for addr: %v failed: %v", builtin.StorageMarketActorAddr, err)
	}

	startID, err := pmas.NextID()
	if err != nil {
		return fmt.Errorf("get startID for deal failed: %v", err)
	}

	nextID, err := mas.NextID()
	if err != nil {
		return fmt.Errorf("get nextID for deal failed: %v", err)
	}

	curProposals, err := mas.Proposals()
	if err != nil {
		return fmt.Errorf("load market proposals failed: %v", err)
	}

	for id := startID; id < nextID; id++ {
		deal, found, err := curProposals.Get(id)
		if err != nil {
			return fmt.Errorf("get deal for id %v failed: %v", id, err)
		}

		if !found {
			return fmt.Errorf("deal id: %v not found", id)
		}

		providerID, err := extract.LookupID(ctx, deal.Provider, ts.Child)
		if err != nil {
			return fmt.Errorf("lookup ID for provider: %v at height %v failed: %v", deal.Provider, ts.Child.Height(), err)
		}
		clientID, err := extract.LookupID(ctx, deal.Client, ts.Child)
		if err != nil {
			return fmt.Errorf("lookup ID for client: %v at height %v failed: %v", deal.Client, ts.Child.Height(), err)
		}

		// get Label bytes after V8
		if av > actors.Version8 {
			dealProposals := []*model.NewDealProposalV8{}
			dealProposal, err := model.NewNewDealProposalV8(id, height, providerID, clientID, *deal)
			if err != nil {
				return fmt.Errorf("new NewDealProposalV8 for dealID: %v failed: %v", id, err)
			}

			dealProposals = append(dealProposals, dealProposal)

			for i := range dealProposals {
				res.Docs = append(res.Docs, dealProposals[i])
			}

		} else {
			dealProposals := []*model.NewDealProposal{}
			dealProposal, _ := model.NewNewDealProposal(id, height, providerID, clientID, *deal)
			dealProposals = append(dealProposals, dealProposal)

			for i := range dealProposals {
				res.Docs = append(res.Docs, dealProposals[i])
			}
		}
	}

	return nil
}
