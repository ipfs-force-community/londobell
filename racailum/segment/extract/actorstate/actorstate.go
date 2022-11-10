package actorstate

import (
	"fmt"
	"log"
	"reflect"

	"github.com/filecoin-project/go-state-types/cbor"
	exported0 "github.com/filecoin-project/specs-actors/actors/builtin/exported"
	"github.com/filecoin-project/specs-actors/actors/runtime"
	exported2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	exported3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/exported"
	exported4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/exported"
	exported5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/exported"
	exported6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/exported"
	exported7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/exported"
	exported8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/exported"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/gen"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/reg"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
)

var builtinActorsCode = make([]cid.Cid, 0)

func init() {
	builtinActors := make([]runtime.VMActor, 0)
	builtinActors = append(builtinActors, exported0.BuiltinActors()...)
	builtinActors = append(builtinActors, exported2.BuiltinActors()...)
	builtinActors = append(builtinActors, exported3.BuiltinActors()...)
	builtinActors = append(builtinActors, exported4.BuiltinActors()...)
	builtinActors = append(builtinActors, exported5.BuiltinActors()...)
	builtinActors = append(builtinActors, exported6.BuiltinActors()...)
	builtinActors = append(builtinActors, exported7.BuiltinActors()...)
	builtinActors = append(builtinActors, exported8.BuiltinActors()...)
	for _, actor := range builtinActors {
		builtinActorsCode = append(builtinActorsCode, actor.Code())
	}
}

var GenRegularHeadID = gen.GenRegularHeadID

// ExtractRegular tries to take all data out of specified actor state head
func ExtractRegular(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead) error {
	return extractState(ctx, res, head, true)
}

func extractState(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, enableActorStateDoc bool) error {
	blkraw, err := ctx.D.ChainBlockstore().Get(ctx.C, head.Head)
	if err != nil {
		return fmt.Errorf("load head block data for %s (%s): %w", head.Addr, head.Head, err)

	}

	// account actor is special
	if builtin.IsAccountActor(head.Code) && enableActorStateDoc {
		as, err := model.NewActorState(head, nil)
		if err != nil {
			return fmt.Errorf("convert actor state from raw for %s (%s): %w", head.Addr, head.Head, err)

		}
		res.Docs = append(res.Docs, as)
		return nil
	}

	actor := head.Actor
	actorVersion, err := actors.VersionForNetwork(ctx.D.GetNetworkVersion(ctx.C, head.Epoch))
	if err != nil {
		return fmt.Errorf("get network.Version for height(%v): %w", head.Epoch, err)
	}

	var realCode cid.Cid
	if actorVersion >= actors.Version8 {
		if actor.Code.String() == "bafk2bzacedwuvyzfaaf6vpxx4lhervvs4qs4ukfqitjxikeemzpec3lbqu5ba" ||
			actor.Code.String() == "bafk2bzacecau3tohdilfx66pohfqdrngpuqd5oew2j5iv3c7sjlrkcm5npqos" ||
			actor.Code.String() == "bafk2bzacedzg2dsdry6cy5nzfldtqatuopljgdxt5hxdwn2gmuj3fk566bndg" {
			fmt.Println("actor.Code.String():", actor.Code.String())
		}

		// todo: 自定义actor
		name := actors.CanonicalName(builtin.ActorNameByCode(head.Code))
		if name == "<unknown>" {
			// todo: 判断actor合理性
			fmt.Printf("unknown actor %v, code: %v\n", actor.Address, actor.Code.String())
		}

		var ok bool
		realCode, ok = actors.GetActorCodeID(actorVersion, name)
		if ok {
			actor.Code = realCode
		}
	}

	var state interface{}
	// 非内置actor  actor->state
	if !IsBuiltinActors(actors.CanonicalName(builtin.ActorNameByCode(head.Code))) {
		// todo: 自定义？？
		if IsCustomActors(actor) {
			// need users to registry
			log.Printf("custom actor skip... actor.Code: %v\n", actor.Code)
			state = nil // todo: 暂时跳过
		} else {
			state, err = reg.DumpExternalActorState(reg.NewExternalActorRegistry(), actor, blkraw.RawData())
			if err != nil {
				return fmt.Errorf("dump external actor state for %s (%s): %w", head.Addr, head.Head, err)
			}
		}
	} else {
		if actorVersion >= actors.Version8 {
			state, err = reg.DumpExternalActorState(reg.NewActorV8Registry(), actor, blkraw.RawData())
			if err != nil {
				return fmt.Errorf("dump v8 builtin actor state for %s (%s): %w", head.Addr, head.Head, err)
			}
		} else {
			state, err = vm.DumpActorState(reg.ActorReg, actor, blkraw.RawData())
			if err != nil {
				return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)
			}
		}
	}

	if gen.IsEmptyState(state) {
		return nil
	}

	raw, ok := state.(cbor.Er)
	if !ok {
		return fmt.Errorf("get non cbor.Er from vm.DumpActorState for %s (%s): %T", head.Addr, head.Head, raw)
	}

	if enableActorStateDoc {
		as, err := model.NewActorState(head, raw)
		if err != nil {
			return fmt.Errorf("convert actor state from raw for %s (%s): %w", head.Addr, head.Head, err)

		}
		res.Docs = append(res.Docs, as)
	}

	rawTyp := reflect.ValueOf(raw).Type()

	exes, ok := reg.Extractors(rawTyp)

	if ok && len(exes) > 0 {
		for ei := range exes {
			if err := exes[ei].Method(ctx, res, head, raw); err != nil {
				return fmt.Errorf("extracting %s: %w", exes[ei].Name, err)
			}
		}
	}

	return nil
}

func getrealcode(code cid.Cid) cid.Cid {
	fmt.Println(code)
	name := actors.CanonicalName(builtin.ActorNameByCode(code))

	var ok bool
	realCode, ok := actors.GetActorCodeID(actors.Version8, name)
	if ok {
		return realCode
	}
	return code
}

func IsBuiltinActors(name string) bool {
	//for _, code := range builtinActorsCode {
	//	if actor.Code == code {
	//		return true
	//	}
	//}

	for _, key := range reg.GetBuiltinActorsKeys() {
		if name == key {
			return true
		}
	}

	return false
}

func IsCustomActors(actor *types.Actor) bool {
	_, _, ok := actors.GetActorMetaByCode(actor.Code)
	return !ok
}
