package actorstate

import (
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("AllocatedSectorsV4", extractAllocatedSectorsV4)
}

func extractAllocatedSectorsV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner4.State) error {
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}
