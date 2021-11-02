package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	power4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/power"
	reward4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/reward"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func init() {
	mustRegisterRegularExtractor("MiningProfitabilityV4", extractMiningProfitabilityV4)
}

// see https://github.com/filecoin-project/specs-actors/blob/v4.0.0/actors/builtin/miner/miner_actor.go#L984-L996
// and https://github.com/filecoin-project/specs-actors/blob/v4.0.0/actors/builtin/miner/monies.go#L155-L181
func extractMiningProfitabilityV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *reward4.State) error {
	blkraw, err := ctx.D.ChainBlockstore().Get(head.Global.Power.Head)
	if err != nil {
		return fmt.Errorf("load head block data for power state (%s): %w", head.Head, err)
	}

	state, err := vm.DumpActorState(ActorReg, head.Global.Power, blkraw.RawData())
	if err != nil {
		return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)
	}

	pwrState, ok := state.(*power4.State)
	if !ok {
		return fmt.Errorf("expecting *power4.State, got %T", pwrState)
	}

	qaPower := miner4.QAPowerForWeight(sectorSize32GiB, 180, big.Zero(), big.Zero())

	storagePledge := miner4.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, miner4.InitialPledgeProjectionPeriod)
	initPledge := miner4.InitialPledgeForPower(qaPower, st.ThisEpochBaselinePower, st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, head.CirculatingSupply.FilCirculating)

	// we just ignore the influence of spaceRacePledgeCap here
	consensusPledge := big.Sub(initPledge, storagePledge)

	detail := model.MiningProfitabilityDetail{
		ExpectedDayReward:         miner4.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, builtin4.EpochsInDay),
		InitialPledge:             initPledge,
		InitialConsensusPledge:    consensusPledge,
		InitialStoragePledge:      storagePledge,
		ProjectionOfInitialPledge: storagePledge, // TODO: projection is just equal to the init storage power here, correct me if I'm wrong
		ProjectionOfFaultFee:      miner4.PledgePenaltyForContinuedFault(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower),
	}

	id, err := GenRegularHeadID(head.Head, head.Addr, head.Epoch)
	if err != nil {
		return fmt.Errorf("gen regular id: %w", err)
	}

	res.Docs = append(res.Docs, &model.MiningProfitability{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    id,
			Path:  []cid.Cid{head.Head, head.Global.Power.Head},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},

		Detail: detail,
	})

	return nil
}
