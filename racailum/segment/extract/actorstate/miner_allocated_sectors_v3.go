package actorstate

import (
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("AllocatedSectorsV3", extractAllocatedSectorsV3)
}

func extractAllocatedSectorsV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner3.State) error {
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}
