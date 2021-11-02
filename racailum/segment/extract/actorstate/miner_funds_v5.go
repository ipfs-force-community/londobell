package actorstate

import (
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV5", extractMinerFundsV5)
}

func extractMinerFundsV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner5.State) error {
	return extractMinerFunds(ctx, res, head, st)
}
