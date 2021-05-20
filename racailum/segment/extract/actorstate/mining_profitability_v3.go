package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	power3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	reward3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
)

func init() {
	mustRegisterRegularExtractor("MiningProfitabilityV3", extractMiningProfitabilityV3)

}

// see https://github.com/filecoin-project/specs-actors/blob/v3.0.3/actors/builtin/miner/miner_actor.go#L976-L988
// and https://github.com/filecoin-project/specs-actors/blob/v3.0.3/actors/builtin/miner/monies.go#L143-L169
func extractMiningProfitabilityV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *reward3.State) error {
	blkraw, err := ctx.D.ChainBlockstore().Get(head.Global.Power.Head)
	if err != nil {
		return fmt.Errorf("load head block data for power state (%s): %w", head.Head, err)
	}

	state, err := vm.DumpActorState(head.Global.Power, blkraw.RawData())
	if err != nil {
		return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)
	}

	pwrState, ok := state.(*power3.State)
	if !ok {
		return fmt.Errorf("expecting *power3.State, got %T", pwrState)
	}

	qaPower := miner3.QAPowerForWeight(sectorSize32GiB, 180, big.Zero(), big.Zero())

	storagePledge := miner3.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, miner3.InitialPledgeProjectionPeriod)
	initPledge := miner3.InitialPledgeForPower(qaPower, st.ThisEpochBaselinePower, st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, head.CirculatingSupply.FilCirculating)

	// we just ignore the influence of spaceRacePledgeCap here
	consensusPledge := big.Sub(initPledge, storagePledge)

	detail := model.MiningProfitabilityDetail{
		ExpectedDayReward:         miner3.ExpectedRewardForPower(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower, builtin3.EpochsInDay),
		InitialPledge:             initPledge,
		InitialConsensusPledge:    consensusPledge,
		InitialStoragePledge:      storagePledge,
		ProjectionOfInitialPledge: storagePledge, // TODO: projection is just equal to the init storage power here, correct me if I'm wrong
		ProjectionOfFaultFee:      miner3.PledgePenaltyForContinuedFault(st.ThisEpochRewardSmoothed, pwrState.ThisEpochQAPowerSmoothed, qaPower),
	}

	id, err := genRegularHeadID(head.Head, head.Addr, head.Epoch)
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
