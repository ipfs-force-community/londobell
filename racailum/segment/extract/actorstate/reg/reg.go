package reg

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

var (
	extractCtxType = reflect.TypeOf((*extract.Ctx)(nil))
	extractResType = reflect.TypeOf((*extract.Res)(nil))
	actorHeadType  = reflect.TypeOf((*common.ActorHead)(nil))
	cborErType     = reflect.TypeOf((*cbor.Er)(nil)).Elem()
	errorType      = reflect.TypeOf((*error)(nil)).Elem()
)

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

type Extractor struct {
	Name     string
	Method   func(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, state cbor.Er) error
	PreCheck func(ctx *extract.Ctx, name string, epoch abi.ChainEpoch) bool
}

type registry struct {
	sync.RWMutex
	e map[reflect.Type][]Extractor
}

func All() []Extractor {
	execs := make([]Extractor, 0, 32)

	extractorRegistry.regular.RLock()
	for _, exts := range extractorRegistry.regular.e {
		execs = append(execs, exts...)
	}
	extractorRegistry.regular.RUnlock()
	return execs
}

func Extractors(typ reflect.Type) ([]Extractor, bool) {
	extractorRegistry.regular.RLock()
	execs, ok := extractorRegistry.regular.e[typ]
	extractorRegistry.regular.RUnlock()

	return execs, ok
}

var extractorRegistry = struct {
	regular *registry
	pre     *preCheck
}{
	regular: &registry{
		e: make(map[reflect.Type][]Extractor),
	},
	pre: &preCheck{
		c: make(map[string]Check),
	},
}

func MustRegisterRegularExtractor(name string, fn interface{}) {
	if err := registerExtractor(extractorRegistry.regular, extractorRegistry.pre, name, fn); err != nil {
		panic(fmt.Errorf("register actor state regulaer extractor: %s", err))
	}
}

func registerExtractor(reg *registry, pre *preCheck, name string, fn interface{}) error {
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
	reg.e[stateInType] = append(reg.e[stateInType], Extractor{
		Name:   name,
		Method: exfn,
		PreCheck: func(ctx *extract.Ctx, name string, epoch abi.ChainEpoch) bool {
			pre.Lock()
			defer pre.Unlock()
			return validate(ctx, pre.c[name], epoch)
		},
	})
	reg.Unlock()

	return nil
}

type Check struct {
	ZeroHourOption func(*extract.Ctx) bool
	TickOption     func(*extract.Ctx) int
}

type preCheck struct {
	sync.RWMutex
	c map[string]Check
}

func MustRegisterPreCheck(name string, getZeroHourOption func(*extract.Ctx) bool, getTickOption func(*extract.Ctx) int) {
	if err := registerPreCheck(extractorRegistry.pre, name, getZeroHourOption, getTickOption); err != nil {
		panic(fmt.Errorf("register actor pre check failed: %s", err))
	}
}

func registerPreCheck(pre *preCheck, name string, getZeroHourOption func(*extract.Ctx) bool, getTickOption func(*extract.Ctx) int) error {
	pre.Lock()
	pre.c[name] = Check{ZeroHourOption: getZeroHourOption, TickOption: getTickOption}
	pre.Unlock()

	return nil
}

func validate(ctx *extract.Ctx, check Check, epoch abi.ChainEpoch) bool {
	isZeroHour := check.ZeroHourOption != nil && common.IsZeroHour(check.ZeroHourOption(ctx), epoch)
	isExtract := check.TickOption == nil || check.TickOption != nil && extract.IsExtract(check.TickOption(ctx), ctx, epoch)

	if !isZeroHour && !isExtract {
		return false
	}

	return true
}
