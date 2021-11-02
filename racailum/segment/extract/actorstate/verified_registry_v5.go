package actorstate

import (
	verifreg5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/verifreg"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("VerifRegV5", extractVerifRegV5)
}

func extractVerifRegV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *verifreg5.State) error {
	return extractVerifReg(ctx, res, head, st)
}
