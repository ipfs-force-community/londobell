package actorstate

// import (
//     "bytes"
//     "fmt"

//     "github.com/filecoin-project/go-address"
//     "github.com/ipfs/go-cid"

//     builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
//     power5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/power"
//     adt5 "github.com/filecoin-project/specs-actors/v5/actors/util/adt"

//     "github.com/ipfs-force-community/londobell/common"
//     "github.com/ipfs-force-community/londobell/racailum/segment/extract"
//     "github.com/ipfs-force-community/londobell/racailum/segment/model"
// )

func init() {
	//mustRegisterRegularExtractor("ClaimedPowerV5", extractClaimedPowerV5)
	//
	//schema.Register(
	//	schema.Model{
	//		Name: "claimed-power-v5",
	//		D: &model.ClaimedPower{
	//			Detail: &power5.Claim{},
	//		},
	//	},
	//)
}

// func extractClaimedPowerV5(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, pst *power5.State) error {
//     claims, err := adt5.AsMap(ctx.D.ActorStore(ctx.C), pst.Claims, builtin5.DefaultHamtBitwidth)
//     if err != nil {
//         return fmt.Errorf("construct adt.Map for Claims in *power3.State: %w", err)
//     }

//     out := &power5.Claim{}
//     var buf bytes.Buffer
//     if err := claims.ForEach(out, func(keystr string) error {
//         detail := *out
//         if isEmptyOrZero(detail.RawBytePower) {
//             return nil
//         }

//         buf.Reset()

//         addr, err := address.NewFromBytes([]byte(keystr))
//         if err != nil {
//             return fmt.Errorf("parse addr from adt.Map key %s: %w", keystr, err)
//         }

//         id, err := genClaimedPowerID(&buf, keystr, &detail)
//         if err != nil {
//             return err
//         }

//         res.Docs = append(res.Docs, &model.ClaimedPower{
//             ActorStateExBasic: model.ActorStateExBasic{
//                 ID:    id,
//                 Path:  []cid.Cid{head.Head, pst.Claims},
//                 Addr:  addr,
//                 Epoch: head.Epoch,
//             },
//             Detail: &detail,
//         })

//         *out = power5.Claim{}
//         return nil
//     }); err != nil {
//         return err
//     }

//     return nil
// }
