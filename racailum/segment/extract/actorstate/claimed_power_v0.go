package actorstate

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"

	power0 "github.com/filecoin-project/specs-actors/actors/builtin/power"
	adt0 "github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("ClaimedPowerV0", extractClaimedPowerV0)

	schema.Register(
		schema.Model{
			Name: "claimed-power-v0",
			D: &model.ClaimedPower{
				Detail: &power0.Claim{},
			},
		},
	)
}

func extractClaimedPowerV0(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *power0.State) error { // nolint: deadcode

	claims, err := adt0.AsMap(ctx.D.ActorStore(ctx.C), pst.Claims)

	if err != nil {
		return fmt.Errorf("construct adt.Map for Claims in *power0.State: %w", err)
	}

	out := &power0.Claim{}
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

		*out = power0.Claim{}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
