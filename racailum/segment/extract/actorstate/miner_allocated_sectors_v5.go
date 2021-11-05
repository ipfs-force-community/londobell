package actorstate

import (
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	// mustRegisterRegularExtractor("AllocatedSectorsV5", extractAllocatedSectorsV5)
}

func extractAllocatedSectorsV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner5.State) error { // nolint: deadcode
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}
