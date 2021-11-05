package actorstate

import (
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	// mustRegisterRegularExtractor("AllocatedSectorsV0", extractAllocatedSectorsV0)
}

func extractAllocatedSectorsV0(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner0.State) error { // nolint: deadcode
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}
