package actorstate

import (
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV2", extractMinerFundsV2)
}

func extractMinerFundsV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner2.State) error {
	return extractMinerFunds(ctx, res, head, st)
}
