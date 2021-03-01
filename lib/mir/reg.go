package mir

import (
	"reflect"
	"sync"
)

type mirIdent = [2]reflect.Type

var reg = registry{
	mirfns:        map[mirIdent]mirrorFunc{},
	fieldMappings: map[mirIdent]fieldMappingRes{},
}

type mirrorFunc func(name string, dst, src reflect.Value) error

type registry struct {
	sync.RWMutex

	mirfns        map[mirIdent]mirrorFunc
	fieldMappings map[mirIdent]fieldMappingRes
}

func (r *registry) lookup(dst, src reflect.Type) (mirrorFunc, bool) {
	r.RLock()
	defer r.RUnlock()

	f, ok := r.mirfns[[2]reflect.Type{dst, src}]
	return f, ok
}

func (r *registry) register(dst, src reflect.Type, mirfn mirrorFunc) {
	r.Lock()
	defer r.Unlock()

	r.mirfns[[2]reflect.Type{dst, src}] = mirfn
}

func (r *registry) lookupFieldMapping(dst, src reflect.Type) (fieldMappingRes, bool) {
	r.RLock()
	defer r.RUnlock()

	mres, ok := r.fieldMappings[mirIdent{dst, src}]
	return mres, ok
}

func (r *registry) registerFieldMapping(dst, src reflect.Type, mres fieldMappingRes) {
	r.Lock()
	defer r.Unlock()

	r.fieldMappings[mirIdent{dst, src}] = mres
}
