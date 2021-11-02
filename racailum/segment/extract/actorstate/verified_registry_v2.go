package actorstate

import (
	verifreg2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/verifreg"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("VerifRegV2", extractVerifRegV2)
}

func extractVerifRegV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *verifreg2.State) error {
	return extractVerifReg(ctx, res, head, st)
}
