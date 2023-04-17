package adapter

import (
	"context"
	"fmt"
	"net/http"

	miner10 "github.com/filecoin-project/go-state-types/builtin/v10/miner"

	"github.com/filecoin-project/go-state-types/manifest"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	miner8 "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	util8 "github.com/filecoin-project/go-state-types/builtin/v8/util"
	adt8 "github.com/filecoin-project/go-state-types/builtin/v8/util/adt"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	util9 "github.com/filecoin-project/go-state-types/builtin/v9/util"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	"github.com/gin-gonic/gin"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/cron"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/rand"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"

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

func getDepositToBurnByCode(ctx context.Context, mact *types.Actor, stor adt.Store, curEpoch abi.ChainEpoch) (abi.TokenAmount, error) {
	version := 0
	if name, av, ok := actors.GetActorMetaByCode(mact.Code); ok {
		if name != manifest.MinerKey {
			return abi.NewTokenAmount(0), fmt.Errorf("actor code is not miner: %s", name)
		}

		switch av {

		case actorstypes.Version8:
			state := &miner8.State{}
			err := stor.Get(ctx, mact.Head, state)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}

			if !state.PreCommittedSectorsCleanUp.Defined() {
				return abi.NewTokenAmount(0), nil
			}
			depositToBurn, err := CleanUpExpiredPreCommits(state, stor, curEpoch)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}
			return depositToBurn, nil
		case actorstypes.Version9:
			state := &miner9.State{}
			err := stor.Get(ctx, mact.Head, state)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}

			if !state.PreCommittedSectorsCleanUp.Defined() {
				return abi.NewTokenAmount(0), nil
			}
			depositToBurn, err := CleanUpExpiredPreCommits(state, stor, curEpoch)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}
			return depositToBurn, nil
		case actorstypes.Version10:
			state := &miner10.State{}
			err := stor.Get(ctx, mact.Head, state)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}

			if !state.PreCommittedSectorsCleanUp.Defined() {
				return abi.NewTokenAmount(0), nil
			}
			depositToBurn, err := CleanUpExpiredPreCommits(state, stor, curEpoch)
			if err != nil {
				return abi.NewTokenAmount(0), err
			}
			return depositToBurn, nil
		}
	}

	switch mact.Code {
	case builtin0.StorageMinerActorCodeID:
		return abi.NewTokenAmount(0), fmt.Errorf("no PreCommittedSectorsCleanUp in state%v", version)
	case builtin2.StorageMinerActorCodeID:
		version = 2
		return abi.NewTokenAmount(0), fmt.Errorf("no PreCommittedSectorsCleanUp in state%v", version)
	case builtin3.StorageMinerActorCodeID:
		version = 3
		return abi.NewTokenAmount(0), fmt.Errorf("no PreCommittedSectorsCleanUp in state%v", version)
	case builtin4.StorageMinerActorCodeID:
		version = 4
		return abi.NewTokenAmount(0), fmt.Errorf("no PreCommittedSectorsCleanUp in state%v", version)
	case builtin5.StorageMinerActorCodeID:
		state := miner5.State{}
		err := stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}

		if !state.PreCommittedSectorsCleanUp.Defined() {
			return abi.NewTokenAmount(0), nil
		}
		depositToBurn, err := state.CleanUpExpiredPreCommits(stor, curEpoch)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}
		return depositToBurn, nil
	case builtin6.StorageMinerActorCodeID:
		state := &miner6.State{}
		err := stor.Get(ctx, mact.Head, state)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}

		if !state.PreCommittedSectorsCleanUp.Defined() {
			return abi.NewTokenAmount(0), nil
		}
		depositToBurn, err := state.CleanUpExpiredPreCommits(stor, curEpoch)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}
		return depositToBurn, nil
	case builtin7.StorageMinerActorCodeID:
		state := &miner7.State{}
		err := stor.Get(ctx, mact.Head, state)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}

		if !state.PreCommittedSectorsCleanUp.Defined() {
			return abi.NewTokenAmount(0), nil
		}
		depositToBurn, err := state.CleanUpExpiredPreCommits(stor, curEpoch)
		if err != nil {
			return abi.NewTokenAmount(0), err
		}
		return depositToBurn, nil
	}

	return abi.NewTokenAmount(0), fmt.Errorf("unknown actor code %s", mact.Code)
}

func CleanUpExpiredPreCommits(state interface{}, store adt.Store, currEpoch abi.ChainEpoch) (depositToBurn abi.TokenAmount, err error) {
	switch state.(type) {
	case *miner8.State:
		st := state.(*miner8.State)
		depositToBurn = abi.NewTokenAmount(0)

		// cleanup expired pre-committed sectors
		cleanUpQ, err := util8.LoadBitfieldQueue(store, st.PreCommittedSectorsCleanUp, st.QuantSpecEveryDeadline(), miner8.PrecommitCleanUpAmtBitwidth)
		if err != nil {
			return depositToBurn, xerrors.Errorf("failed to load sector expiry queue: %w", err)
		}

		sectors, modified, err := PopUntil(cleanUpQ, currEpoch)
		if err != nil {
			return depositToBurn, xerrors.Errorf("failed to pop expired sectors: %w", err)
		}

		if modified {
			st.PreCommittedSectorsCleanUp, err = cleanUpQ.Root()
			if err != nil {
				return depositToBurn, xerrors.Errorf("failed to save pre commit clean up queue: %w", err)
			}
		}

		var precommitsToDelete []abi.SectorNumber
		if err = sectors.ForEach(func(i uint64) error {
			sectorNo := abi.SectorNumber(i)
			sector, found, err := st.GetPrecommittedSector(store, sectorNo)
			if err != nil {
				return err
			}
			if !found {
				// already committed/deleted
				return nil
			}

			// mark it for deletion
			precommitsToDelete = append(precommitsToDelete, sectorNo)

			// increment deposit to burn
			depositToBurn = big.Add(depositToBurn, sector.PreCommitDeposit)
			return nil
		}); err != nil {
			return big.Zero(), xerrors.Errorf("failed to check pre-commit expiries: %w", err)
		}

		// Actually delete it.
		if len(precommitsToDelete) > 0 {
			if err := DeletePrecommittedSectors(st, store, precommitsToDelete...); err != nil {
				return big.Zero(), fmt.Errorf("failed to delete pre-commits: %w", err)
			}
		}

		st.PreCommitDeposits = big.Sub(st.PreCommitDeposits, depositToBurn)
		if st.PreCommitDeposits.LessThan(big.Zero()) {
			return big.Zero(), xerrors.Errorf("pre-commit clean up caused negative deposits: %v", st.PreCommitDeposits)
		}

		// This deposit was locked separately to pledge collateral so there's no pledge change here.
		return depositToBurn, nil
	case *miner9.State:
		st := state.(*miner9.State)
		depositToBurn = abi.NewTokenAmount(0)

		// cleanup expired pre-committed sectors
		cleanUpQ, err := util9.LoadBitfieldQueue(store, st.PreCommittedSectorsCleanUp, st.QuantSpecEveryDeadline(), miner9.PrecommitCleanUpAmtBitwidth)
		if err != nil {
			return depositToBurn, xerrors.Errorf("failed to load sector expiry queue: %w", err)
		}

		sectors, modified, err := PopUntil(cleanUpQ, currEpoch)
		if err != nil {
			return depositToBurn, xerrors.Errorf("failed to pop expired sectors: %w", err)
		}

		if modified {
			st.PreCommittedSectorsCleanUp, err = cleanUpQ.Root()
			if err != nil {
				return depositToBurn, xerrors.Errorf("failed to save pre commit clean up queue: %w", err)
			}
		}

		var precommitsToDelete []abi.SectorNumber
		if err = sectors.ForEach(func(i uint64) error {
			sectorNo := abi.SectorNumber(i)
			sector, found, err := st.GetPrecommittedSector(store, sectorNo)
			if err != nil {
				return err
			}
			if !found {
				// already committed/deleted
				return nil
			}

			// mark it for deletion
			precommitsToDelete = append(precommitsToDelete, sectorNo)

			// increment deposit to burn
			depositToBurn = big.Add(depositToBurn, sector.PreCommitDeposit)
			return nil
		}); err != nil {
			return big.Zero(), xerrors.Errorf("failed to check pre-commit expiries: %w", err)
		}

		// Actually delete it.
		if len(precommitsToDelete) > 0 {
			if err := DeletePrecommittedSectors(st, store, precommitsToDelete...); err != nil {
				return big.Zero(), fmt.Errorf("failed to delete pre-commits: %w", err)
			}
		}

		st.PreCommitDeposits = big.Sub(st.PreCommitDeposits, depositToBurn)
		if st.PreCommitDeposits.LessThan(big.Zero()) {
			return big.Zero(), xerrors.Errorf("pre-commit clean up caused negative deposits: %v", st.PreCommitDeposits)
		}

		// This deposit was locked separately to pledge collateral so there's no pledge change here.
		return depositToBurn, nil
	default:
		return big.Zero(), fmt.Errorf("CleanUpExpiredPreCommits gets invalid type: %v", state)
	}
}

func PopUntil(queue interface{}, until abi.ChainEpoch) (values bitfield.BitField, modified bool, err error) {
	var poppedValues []bitfield.BitField
	var poppedKeys []uint64

	stopErr := fmt.Errorf("stop")
	switch queue.(type) {
	case util8.BitfieldQueue:
		q := queue.(util8.BitfieldQueue)
		if err = q.ForEach(func(epoch abi.ChainEpoch, bf bitfield.BitField) error {
			if epoch > until {
				return stopErr
			}
			poppedKeys = append(poppedKeys, uint64(epoch))
			poppedValues = append(poppedValues, bf)
			return err
		}); err != nil && err != stopErr {
			return bitfield.BitField{}, false, err
		}

		// Nothing expired.
		if len(poppedKeys) == 0 {
			return bitfield.New(), false, nil
		}

		if err = q.BatchDelete(poppedKeys, true); err != nil {
			return bitfield.BitField{}, false, err
		}
		merged, err := bitfield.MultiMerge(poppedValues...)
		if err != nil {
			return bitfield.BitField{}, false, err
		}

		return merged, true, nil
	case util9.BitfieldQueue:
		q := queue.(util9.BitfieldQueue)
		if err = q.ForEach(func(epoch abi.ChainEpoch, bf bitfield.BitField) error {
			if epoch > until {
				return stopErr
			}
			poppedKeys = append(poppedKeys, uint64(epoch))
			poppedValues = append(poppedValues, bf)
			return err
		}); err != nil && err != stopErr {
			return bitfield.BitField{}, false, err
		}

		// Nothing expired.
		if len(poppedKeys) == 0 {
			return bitfield.New(), false, nil
		}

		if err = q.BatchDelete(poppedKeys, true); err != nil {
			return bitfield.BitField{}, false, err
		}
		merged, err := bitfield.MultiMerge(poppedValues...)
		if err != nil {
			return bitfield.BitField{}, false, err
		}

		return merged, true, nil
	default:
		return bitfield.BitField{}, false, fmt.Errorf("PopUntil gets invalid type: %v", queue)
	}
}

func DeletePrecommittedSectors(state interface{}, store adt.Store, sectorNos ...abi.SectorNumber) error {
	switch state.(type) {
	case *miner8.State:
		st := state.(*miner8.State)
		precommitted, err := adt8.AsMap(store, st.PreCommittedSectors, sbuiltin.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		for _, sectorNo := range sectorNos {
			err = precommitted.Delete(miner8.SectorKey(sectorNo))
			if err != nil {
				return xerrors.Errorf("failed to delete precommitment for %v: %w", sectorNo, err)
			}
		}
		st.PreCommittedSectors, err = precommitted.Root()
		return err
	case *miner9.State:
		st := state.(*miner9.State)
		precommitted, err := adt9.AsMap(store, st.PreCommittedSectors, sbuiltin.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		for _, sectorNo := range sectorNos {
			err = precommitted.Delete(miner9.SectorKey(sectorNo))
			if err != nil {
				return xerrors.Errorf("failed to delete precommitment for %v: %w", sectorNo, err)
			}
		}
		st.PreCommittedSectors, err = precommitted.Root()
		return err
	default:
		return fmt.Errorf("DeletePrecommittedSectors gets invalid type: %v", state)
	}
}
