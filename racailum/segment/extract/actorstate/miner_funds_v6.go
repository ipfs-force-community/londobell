package actorstate

import (
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV6", extractMinerFundsV6)
}

func extractMinerFundsV6(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner6.State) error {
	return extractMinerFunds(ctx, res, head, st)
}
