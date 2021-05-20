package actorstate

import (
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("AllocatedSectorsV2", extractAllocatedSectorsV2)
}

func extractAllocatedSectorsV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner2.State) error {
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}
