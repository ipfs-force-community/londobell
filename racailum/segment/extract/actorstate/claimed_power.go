package actorstate

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func init() {
	mustRegisterRegularExtractor("ClaimedPowerV2", extractClaimedPowerV2)
	mustRegisterRegularExtractor("ClaimedPowerV3", extractClaimedPowerV3)

	schema.Register(
		schema.Model{
			Name: "claimed-power-v2",
			D: &model.ClaimedPower{
				Detail: &power2.Claim{},
			},
		},

		schema.Model{
			Name: "claimed-power-v3",
			D: &model.ClaimedPower{
				Detail: &power3.Claim{},
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

func extractClaimedPowerV3(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *power3.State) error {
	claims, err := adt3.AsMap(ctx.D.ActorStore(ctx.C), pst.Claims, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return fmt.Errorf("construct adt.Map for Claims in *power3.State: %w", err)
	}

	out := &power3.Claim{}
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

		*out = power3.Claim{}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func genClaimedPowerID(buf *bytes.Buffer, keystr string, detail cbor.Er) (cid.Cid, error) {
	if _, err := buf.WriteString(keystr); err != nil {
		return cid.Undef, fmt.Errorf("write key string %s: %w", keystr, err)
	}

	if err := detail.MarshalCBOR(buf); err != nil {
		return cid.Undef, fmt.Errorf("write claim data for %s: %w", keystr, err)
	}

	id, err := common.CidBuilder.Sum(buf.Bytes())
	if err != nil {
		return cid.Undef, fmt.Errorf("construct cid for %s: %w", keystr, err)
	}

	return id, nil
}
