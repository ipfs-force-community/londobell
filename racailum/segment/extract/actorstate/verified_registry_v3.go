package actorstate

import (
	verifreg3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/verifreg"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("VerifRegV3", extractVerifRegV3)
}

func extractVerifRegV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *verifreg3.State) error {
	return extractVerifReg(ctx, res, head, st)
}
