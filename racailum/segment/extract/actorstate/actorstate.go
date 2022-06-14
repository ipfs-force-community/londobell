package actorstate

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/gen"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/reg"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

var GenRegularHeadID = gen.GenRegularHeadID
var log = logging.Logger("actorstate")

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
		name := actors.CanonicalName(builtin.ActorNameByCode(head.Code))

		var ok bool
		realCode, ok = actors.GetActorCodeID(actorVersion, name)
		if ok {
			actor.Code = realCode
		}

		log.Infow("update code", "head.Code: %v", head.Code, "actor.Code: %v", actor.Code, "name: %v", name)
	}

	state, err := vm.DumpActorState(reg.ActorReg, actor, blkraw.RawData())
	if err != nil {
		return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)

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
