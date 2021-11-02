package actorstate

import (
	multisig4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/multisig"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

func init() {
	mustRegisterRegularExtractor("MultisigBalanceV4", extractMultisigBalanceV4)
}

func extractMultisigBalanceV4(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, mst *multisig4.State) error {
	return extractMultisigBalanceDetail(ctx, res, head, mst)
}
