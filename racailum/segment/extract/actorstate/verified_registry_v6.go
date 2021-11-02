package actorstate

import (
	verifreg6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/verifreg"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("VerifRegV6", extractVerifRegV6)
}

func extractVerifRegV6(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st *verifreg6.State) error {
	return extractVerifReg(ctx, res, head, st)
}
