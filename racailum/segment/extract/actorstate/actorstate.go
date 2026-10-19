package actorstate

import (
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/gen"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract/actorstate/reg"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

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
	// 解码依据必须是 state tree 里真实的 actor code：
	// 快照 label(head.Epoch) 与真实 state(ParentState) 在升级边界会跨版本
	// （NV29 激活那一帧即如此），用 label 换算出的 code 覆盖真实 code
	// 会导致 CBOR 字段数不符（v18/v19 reward state 差 3 个字段）。
	state, err := vm.DumpActorState(consensus.NewTipSetExecutor(filcns.RewardFunc).NewActorRegistry(), actor, blkraw.RawData())
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
			if !exes[ei].PreCheck(ctx, exes[ei].Name, head.Epoch) {
				continue
			}

			if err := exes[ei].Method(ctx, res, head, raw); err != nil {
				return fmt.Errorf("extracting %s: %w", exes[ei].Name, err)
			}
		}
	}

	return nil
}
