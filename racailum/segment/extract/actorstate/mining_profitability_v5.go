package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	power5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/power"
	reward5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/reward"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
)

func init() {
	mustRegisterRegularExtractor("MiningProfitabilityV5", extractMiningProfitabilityV5)
}

// see https://github.com/filecoin-project/specs-actors/blob/v4.0.0/actors/builtin/miner/miner_actor.go#L984-L996
// and https://github.com/filecoin-project/specs-actors/blob/v4.0.0/actors/builtin/miner/monies.go#L155-L181
func extractMiningProfitabilityV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *reward5.State) error {
	blkraw, err := ctx.D.ChainBlockstore().Get(head.Global.Power.Head)
	if err != nil {
		return fmt.Errorf("load head block data for power state (%s): %w", head.Head, err)
	}

	state, err := vm.DumpActorState(ActorReg, head.Global.Power, blkraw.RawData())
	if err != nil {
		return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)
	}

	pwrState, ok := state.(*power5.State)
	if !ok {
		return fmt.Errorf("expecting *power5.State, got %T", pwrState)
	}

	qaPower := miner5.QAPowerForWeight(sectorSize32GiB, 180, big.Zero(), big.Zero())

	storagePledge := miner5.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, miner5.InitialPledgeProjectionPeriod)
	initPledge := miner5.InitialPledgeForPower(qaPower, st.ThisEpochBaselinePower, st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, head.CirculatingSupply.FilCirculating)

	// we just ignore the influence of spaceRacePledgeCap here
	consensusPledge := big.Sub(initPledge, storagePledge)

	detail := model.MiningProfitabilityDetail{
		ExpectedDayReward:         miner5.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, builtin5.EpochsInDay),
		InitialPledge:             initPledge,
		InitialConsensusPledge:    consensusPledge,
		InitialStoragePledge:      storagePledge,
		ProjectionOfInitialPledge: storagePledge, // TODO: projection is just equal to the init storage power here, correct me if I'm wrong
		ProjectionOfFaultFee:      miner5.PledgePenaltyForContinuedFault(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower),
		Mined:                     st.TotalStoragePowerReward,
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
