package actorstate

import (
	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV2", extractMultisigBalanceV2)
}

func extractMultisigBalanceV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig2.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}
