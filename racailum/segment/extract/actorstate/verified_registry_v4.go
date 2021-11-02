package actorstate

import (
	verifreg4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/verifreg"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("VerifRegV4", extractVerifRegV4)
}

func extractVerifRegV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *verifreg4.State) error {
	return extractVerifReg(ctx, res, head, st)
}
