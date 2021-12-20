package reg

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
)

type ExtractorMethod func(ctx *extract.Ctx, res *extract.Res, head *common.ActorHead, state cbor.Er) error

type Extractor struct {
	Name   string
	Method ExtractorMethod
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
}{
	regular: &registry{
		e: make(map[reflect.Type][]Extractor),
	},
}

func MustRegisterRegularExtractor(name string, stateInType reflect.Type, fn ExtractorMethod) {
	if err := registerExtractor(extractorRegistry.regular, name, stateInType, fn); err != nil {
		panic(fmt.Errorf("register actor state regulaer extractor: %s", err))
	}
}

func registerExtractor(reg *registry, name string, stateInType reflect.Type, fn ExtractorMethod) error {
	reg.Lock()
	reg.e[stateInType] = append(reg.e[stateInType], Extractor{
		Name:   name,
		Method: fn,
	})
	reg.Unlock()
	return nil
}
