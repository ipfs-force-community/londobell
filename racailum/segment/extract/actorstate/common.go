package actorstate

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/vm"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/racailum/segment/extract"
	"github.com/dtynn/londobell/racailum/segment/model"
)

var (
	extractCtxType = reflect.TypeOf((*extract.Ctx)(nil))
	extractResType = reflect.TypeOf((*extract.Res)(nil))
	actorHeadType  = reflect.TypeOf((*common.ActorHead)(nil))
	cborErType     = reflect.TypeOf((*cbor.Er)(nil)).Elem()
	errorType      = reflect.TypeOf((*error)(nil)).Elem()
)

var (
	bigZero         = big.Zero()
	tokenAmountZero = abi.NewTokenAmount(0)
)

type registry struct {
	sync.RWMutex
	e map[reflect.Type][]extractor
}

var ActorReg = filcns.NewActorRegistry()

var extractorRegistry = struct {
	regular *registry
}{
	regular: &registry{
		e: make(map[reflect.Type][]extractor),
	},
}

type extractor struct {
	name   string
	method func(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, state cbor.Er) error
}

var expectedCommonInTypes = []reflect.Type{
	extractCtxType,
	extractResType,
	actorHeadType,
}

var (
	expectedCommonInTypesCount = len(expectedCommonInTypes)
	expectedNumIn              = expectedCommonInTypesCount + 1
	stateInIndex               = expectedCommonInTypesCount
)

func mustRegisterRegularExtractor(name string, fn interface{}) {
	if err := registerExtractor(extractorRegistry.regular, name, fn); err != nil {
		panic(fmt.Errorf("register actor state regulaer extractor: %s", err))
	}
}

func registerExtractor(reg *registry, name string, fn interface{}) error {
	fnVal := reflect.ValueOf(fn)
	if kind := fnVal.Kind(); kind != reflect.Func {
		return fmt.Errorf("actor state-ex extractor should be a func, got %s", kind)
	}

	fnTyp := fnVal.Type()
	if numIn := fnTyp.NumIn(); numIn != expectedNumIn {
		return fmt.Errorf("actor state-ex extractor should have %d inputs, got %d", expectedNumIn, numIn)
	}

	if numOut := fnTyp.NumOut(); numOut != 1 {
		return fmt.Errorf("actor state-ex extractor should have 1 output, got %d", numOut)
	}

	for inIdx := range expectedCommonInTypes {
		if inType := fnTyp.In(inIdx); inType != expectedCommonInTypes[inIdx] {
			return fmt.Errorf("#%d input should be %s, got %s", inIdx, expectedCommonInTypes[inIdx], inType)
		}
	}

	stateInType := fnTyp.In(stateInIndex)
	if !stateInType.Implements(cborErType) {
		return fmt.Errorf("input of state should implement cbor.Er, got %s", stateInType)
	}

	if outType := fnTyp.Out(0); outType != errorType && !outType.Implements(errorType) {
		return fmt.Errorf("output should be error or implement error, got %s", outType)
	}

	exfn := func(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, state cbor.Er) error {
		if state == nil {
			return fmt.Errorf("expecting %s, got nil", stateInType)
		}

		outs := fnVal.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(res),
			reflect.ValueOf(head),
			reflect.ValueOf(state),
		})

		out := outs[0]
		if out.IsNil() {
			return nil
		}

		return out.Interface().(error)
	}

	reg.Lock()
	reg.e[stateInType] = append(reg.e[stateInType], extractor{
		name:   name,
		method: exfn,
	})
	reg.Unlock()

	return nil
}

// ExtractRegular tries to take all data out of specified actor state head
func ExtractRegular(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead) error {
	return extractState(ctx, res, head, extractorRegistry.regular, true)
}

func extractState(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, reg *registry, enableActorStateDoc bool) error {
	blkraw, err := ctx.D.ChainBlockstore().Get(head.Head)
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

	state, err := vm.DumpActorState(ActorReg, head.Actor, blkraw.RawData())
	if err != nil {
		return fmt.Errorf("dump actor state for %s (%s): %w", head.Addr, head.Head, err)

	}

	if isEmptyState(state) {
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
	reg.RLock()
	exes, ok := reg.e[rawTyp]
	reg.RUnlock()

	if ok && len(exes) > 0 {
		for ei := range exes {
			if err := exes[ei].method(ctx, res, head, raw); err != nil {
				return fmt.Errorf("extracting %s: %w", exes[ei].name, err)
			}
		}
	}

	return nil
}

func GenRegularHeadID(root cid.Cid, addr address.Address, epoch abi.ChainEpoch) (cid.Cid, error) {
	rbytes := root.Bytes()
	abytes := addr.Bytes()

	rbytesSize := len(rbytes)
	bytesSize := rbytesSize + len(abytes)
	totalSize := bytesSize + 8

	payload := make([]byte, totalSize)
	copy(payload[:rbytesSize], rbytes)
	copy(payload[rbytesSize:bytesSize], abytes)
	binary.BigEndian.PutUint64(payload[bytesSize:], uint64(epoch))

	return common.CidBuilder.Sum(payload)
}

func isEmptyOrZero(n big.Int) bool {
	return n.Int == nil || n.Equals(bigZero)
}
