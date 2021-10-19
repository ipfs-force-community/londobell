package actorstate

import (
	multisig6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/multisig"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV6", extractMultisigBalanceV6)
}

func extractMultisigBalanceV6(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig6.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}
