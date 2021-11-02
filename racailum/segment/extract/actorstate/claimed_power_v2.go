package actorstate

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"

	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("ClaimedPowerV2", extractClaimedPowerV2)

	schema.Register(
		schema.Model{
			Name: "claimed-power-v2",
			D: &model.ClaimedPower{
				Detail: &power2.Claim{},
			},
		},
	)
}

func extractClaimedPowerV2(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *power2.State) error {
	claims, err := adt2.AsMap(ctx.D.ActorStore(ctx.C), pst.Claims)
	if err != nil {
		return fmt.Errorf("construct adt.Map for Claims in *power2.State: %w", err)
	}

	out := &power2.Claim{}
	var buf bytes.Buffer
	if err := claims.ForEach(out, func(keystr string) error {
		detail := *out
		if isEmptyOrZero(detail.RawBytePower) {
			return nil
		}

		buf.Reset()

		addr, err := address.NewFromBytes([]byte(keystr))
		if err != nil {
			return fmt.Errorf("parse addr from adt.Map key %s: %w", keystr, err)
		}

		id, err := genClaimedPowerID(&buf, keystr, &detail)
		if err != nil {
			return err
		}

		res.Docs = append(res.Docs, &model.ClaimedPower{
			ActorStateExBasic: model.ActorStateExBasic{
				ID:    id,
				Path:  []cid.Cid{head.Head, pst.Claims},
				Addr:  addr,
				Epoch: head.Epoch,
			},
			Detail: &detail,
		})

		*out = power2.Claim{}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
