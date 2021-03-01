package actorstate

import (
	"github.com/filecoin-project/go-bitfield"
	"github.com/ipfs/go-cid"

	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"

	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("AllocatedSectorsV2", extractAllocatedSectorsV2)
	mustRegisterRegularExtractor("AllocatedSectorsV3", extractAllocatedSectorsV3)

	schema.Register(
		schema.Model{
			Name: "allocated-sectors",
			D:    &model.AllocatedSectors{},
		},
	)
}

func extractAllocatedSectorsV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner2.State) error {
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}

func extractAllocatedSectorsV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *miner3.State) error {
	return extractAllocatedSectors(ctx, res, head, pst.AllocatedSectors)
}

func extractAllocatedSectors(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, root cid.Cid) error {
	// ignore empty
	if root.Equals(emptyMinerStateV2.AllocatedSectors) || root.Equals(emptyMinerStateV3.AllocatedSectors) {
		return nil
	}

	var allocatedSectors bitfield.BitField
	if err := extractCborObject(ctx.D, root, &allocatedSectors); err != nil {
		return err

	}

	detail, err := model.NewBitfieldDetail(allocatedSectors, false)
	if err != nil {
		return err
	}

	res.Docs = append(res.Docs, &model.AllocatedSectors{
		ActorStateExBasic: model.ActorStateExBasic{
			ID:    root,
			Path:  []cid.Cid{head.Head, root},
			Addr:  head.Addr,
			Epoch: head.Epoch,
		},
		Detail: detail,
	})
	return nil

}
