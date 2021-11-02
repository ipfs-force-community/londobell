package actorstate

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	verifreg2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/verifreg"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	verifreg3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/verifreg"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	verifreg4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/verifreg"
	adt4 "github.com/filecoin-project/specs-actors/v4/actors/util/adt"

	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	verifreg5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/verifreg"
	adt5 "github.com/filecoin-project/specs-actors/v5/actors/util/adt"

	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	verifreg6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/verifreg"
	adt6 "github.com/filecoin-project/specs-actors/v6/actors/util/adt"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	schema.Register(
		schema.Model{
			Name: "verifreg",
			D:    &model.VerifiedRegistry{},
		},
	)
}

type adtMap interface {
	ForEach(out cbor.Unmarshaler, fn func(key string) error) error
}

type namedAdtMapRoot struct {
	name string
	root cid.Cid
}

func extractVerifReg(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, st interface{}) error {
	if ticks := ctx.Opts.StateRegular.VerifRegTicks; ticks > 0 && head.Epoch%(abi.ChainEpoch(ticks)*ctx.Opts.StateRegular.Interval) != 0 {
		return nil
	}

	var mapRoots []namedAdtMapRoot
	var mapConstructor func(cid.Cid) (adtMap, error)

	switch st := st.(type) {
	case *verifreg2.State:
		mapRoots = []namedAdtMapRoot{
			{
				name: "Verifier",
				root: st.Verifiers,
			},
			{
				name: "VerifiedClient",
				root: st.VerifiedClients,
			},
		}

		mapConstructor = func(c cid.Cid) (adtMap, error) {
			return adt2.AsMap(ctx.D.ActorStore(ctx.C), c)
		}

	case *verifreg3.State:
		mapRoots = []namedAdtMapRoot{
			{
				name: "Verifier",
				root: st.Verifiers,
			},
			{
				name: "VerifiedClient",
				root: st.VerifiedClients,
			},
		}

		mapConstructor = func(c cid.Cid) (adtMap, error) {
			return adt3.AsMap(ctx.D.ActorStore(ctx.C), c, builtin3.DefaultHamtBitwidth)
		}

	case *verifreg4.State:
		mapRoots = []namedAdtMapRoot{
			{
				name: "Verifier",
				root: st.Verifiers,
			},
			{
				name: "VerifiedClient",
				root: st.VerifiedClients,
			},
		}

		mapConstructor = func(c cid.Cid) (adtMap, error) {
			return adt4.AsMap(ctx.D.ActorStore(ctx.C), c, builtin4.DefaultHamtBitwidth)
		}
	case *verifreg5.State:
		mapRoots = []namedAdtMapRoot{
			{
				name: "Verifier",
				root: st.Verifiers,
			},
			{
				name: "VerifiedClient",
				root: st.VerifiedClients,
			},
		}

		mapConstructor = func(c cid.Cid) (adtMap, error) {
			return adt5.AsMap(ctx.D.ActorStore(ctx.C), c, builtin5.DefaultHamtBitwidth)
		}
	case *verifreg6.State:
		mapRoots = []namedAdtMapRoot{
			{
				name: "Verifier",
				root: st.Verifiers,
			},
			{
				name: "VerifiedClient",
				root: st.VerifiedClients,
			},
		}

		mapConstructor = func(c cid.Cid) (adtMap, error) {
			return adt6.AsMap(ctx.D.ActorStore(ctx.C), c, builtin6.DefaultHamtBitwidth)
		}
	default:
		return fmt.Errorf("unexpected state: %T", st)
	}

	for mi := range mapRoots {
		root := mapRoots[mi].root
		name := mapRoots[mi].name

		m, err := mapConstructor(root)
		if err != nil {
			return fmt.Errorf("construct adt.Map: %w", err)
		}

		out := &abi.StoragePower{}
		if err := m.ForEach(out, func(keystr string) error {
			datacap := *out
			if isEmptyOrZero(datacap) {
				return nil
			}

			addr, err := address.NewFromBytes([]byte(keystr))
			if err != nil {
				return fmt.Errorf("parse addr from adt.Map key %s: %w", keystr, err)
			}

			id, err := GenRegularHeadID(head.Head, addr, head.Epoch)
			if err != nil {
				return err
			}

			res.Docs = append(res.Docs, &model.VerifiedRegistry{
				ActorStateExBasic: model.ActorStateExBasic{
					ID:    id,
					Path:  []cid.Cid{head.Head, root},
					Addr:  addr,
					Epoch: head.Epoch,
				},
				Detail: model.VerifiedRegistryDetail{
					Type: name,
					Cap:  datacap,
				},
			})

			*out = abi.StoragePower{}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}
