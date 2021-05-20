package actorstate

import (
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV3", extractMinerFundsV3)

}

func extractMinerFundsV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner3.State) error {
	return extractMinerFunds(ctx, res, head, st)
}
