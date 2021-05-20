package actorstate

import (
	multisig3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV3", extractMultisigBalanceV3)

}

func extractMultisigBalanceV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig3.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}
