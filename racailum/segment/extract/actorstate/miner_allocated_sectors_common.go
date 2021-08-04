package actorstate

import (
	"github.com/filecoin-project/go-bitfield"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	schema.Register(
		schema.Model{
			Name: "allocated-sectors",
			D:    &model.AllocatedSectors{},
		},
	)
}

func extractAllocatedSectors(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, root cid.Cid) error {
	// ignore empty
	if root.Equals(emptyMinerStateV2.AllocatedSectors) || root.Equals(emptyMinerStateV3.AllocatedSectors) ||
		root.Equals(emptyMinerStateV4.AllocatedSectors) || root.Equals(emptyMinerStateV5.AllocatedSectors) {
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
