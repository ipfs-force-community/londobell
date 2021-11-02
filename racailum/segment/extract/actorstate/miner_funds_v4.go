package actorstate

import (
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MinerFundsV4", extractMinerFundsV4)
}

func extractMinerFundsV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *miner4.State) error {
	return extractMinerFunds(ctx, res, head, st)
}
