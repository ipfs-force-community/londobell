package adapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/cron"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/gin-gonic/gin"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetPreCommitDepositToBurnInfo(c *gin.Context) {
	alog := log.With("method", "GetPreCommitDepositToBurnInfo")
	req := model.PreCommitDepositToBurnReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		pstate           cid.Cid
		vmCron           vm.Interface
		allMiners        []address.Address
		depositToBurnRes []model.PreCommitDepositToBurnRes
		baseFee          abi.TokenAmount
		parentTs         *types.TipSet
	)

	if req.Epoch == 0 {
		util.ReturnOnErr(c, alog, fmt.Errorf("epoch must be specified"))
		return
	}

	curEpoch := abi.ChainEpoch(req.Epoch)
	curTs, err := Components.Full.ChainGetTipSetByHeight(ctx, curEpoch, types.EmptyTSK)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	sm, ok := Components.SM.(*stmgr.StateManager)
	if !ok {
		util.ReturnOnErr(c, alog, fmt.Errorf("Components.SM is not *stmgr.StateManager type"))
		return
	}

	r := rand.NewStateRand(sm.ChainStore(), curTs.Cids(), sm.Beacon(), sm.GetNetworkVersion)
	makeVMWithBaseStateAndEpoch := func(base cid.Cid, e abi.ChainEpoch) (vm.Interface, error) {
		vmopt := &vm.VMOpts{
			StateBase:      base,
			Epoch:          e,
			Rand:           r,
			Bstore:         sm.ChainStore().StateBlockstore(),
			Actors:         filcns.NewActorRegistry(),
			Syscalls:       sm.Syscalls,
			CircSupplyCalc: sm.GetVMCirculatingSupply,
			NetworkVersion: sm.GetNetworkVersion(ctx, e),
			BaseFee:        baseFee,
			LookbackState:  stmgr.LookbackStateGetterForTipset(sm, curTs),
		}

		return sm.VMConstructor()(ctx, vmopt)
	}

	runCron := func(vmCron vm.Interface, epoch abi.ChainEpoch) error {
		cronMsg := &types.Message{
			To:         cron.Address,
			From:       builtin.SystemActorAddr,
			Nonce:      uint64(epoch),
			Value:      types.NewInt(0),
			GasFeeCap:  types.NewInt(0),
			GasPremium: types.NewInt(0),
			GasLimit:   build.BlockGasLimit * 10000, // Make super sure this is never too little
			Method:     cron.Methods.EpochTick,
			Params:     nil,
		}
		log.Infof("(sm *StateManager) ApplyBlocks runCron begin, height: %v", epoch)

		ret, err := vmCron.ApplyImplicitMessage(ctx, cronMsg)

		log.Infof("(sm *StateManager) ApplyBlocks runCron end, height: %v", epoch)

		if err != nil {
			return fmt.Errorf("running cron: %w", err)
		}

		if ret.ExitCode != 0 {
			return fmt.Errorf("cron exit was non-zero: %d", ret.ExitCode)
		}

		return nil
	}

	// curEpoch is null round
	if curTs.Height() < curEpoch {
		var msgs []*types.Message
		pstate, _, err = stmgr.ComputeState(ctx, sm, curTs.Height(), msgs, curTs)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		parentTs = curTs
		baseFee, err = Components.CS.ComputeBaseFee(ctx, curTs)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		log.Infof("state at height %v is %v", curTs.Height(), pstate)
	} else {
		pstate = curTs.Blocks()[0].ParentStateRoot
		parentTs, err = Components.CS.LoadTipSet(ctx, curTs.Parents())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		baseFee = curTs.Blocks()[0].ParentBaseFee

		log.Infof("state at height %v is %v", parentTs.Height(), pstate)
	}

	// run cron for null rounds
	for i := parentTs.Height(); i < curEpoch; i++ {
		if i > parentTs.Height() {
			vmCron, err = makeVMWithBaseStateAndEpoch(pstate, i)
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}

			if err = runCron(vmCron, i); err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
			pstate, err = vmCron.Flush(ctx)
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
		}

		pstate, err = sm.HandleStateForks(ctx, pstate, i, nil, curTs)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		log.Infof("state at height %v is %v", i, pstate)
	}

	parentSt, err := sm.StateTree(pstate)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	stor := sm.ChainStore().ActorStore(ctx)

	for _, addr := range req.Miners {
		miner, err := address.NewFromString(addr)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		allMiners = append(allMiners, miner)
	}

	// todo: 限制allMiners数量
	for _, miner := range allMiners {
		mact, err := parentSt.GetActor(miner)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		depositToBurn, err := getDepositToBurnByCode(ctx, mact, stor, curEpoch)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		if depositToBurn.IsZero() {
			continue
		}
		depositToBurnRes = append(depositToBurnRes, model.PreCommitDepositToBurnRes{Miner: miner, Epoch: curEpoch, DepositToBurn: depositToBurn})
	}

	res.Data = depositToBurnRes
	c.JSON(http.StatusOK, res)
}
