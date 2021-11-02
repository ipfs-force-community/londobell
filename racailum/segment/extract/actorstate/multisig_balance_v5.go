package actorstate

import (
	multisig5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV5", extractMultisigBalanceV5)
}

func extractMultisigBalanceV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig5.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}
