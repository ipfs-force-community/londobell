package actor

import (
	"reflect"

	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/filecoin-project/lotus/chain/vm"
)

// MethodSend is the method info for builtin.MethodSend
var MethodSend = MethodInfo{
	Actor: "",
	Method: vm.MethodMeta{
		Name: "Send",
	},
}

// MethodInfo includes actor name & method meta
type MethodInfo struct {
	Actor  string
	Method vm.MethodMeta
}

// ParamObj returns a new instance of param object
func (mi *MethodInfo) ParamObj() cbor.Er {
	if mi.Method.Params == nil {
		return nil
	}

	return reflect.New(mi.Method.Params.Elem()).Interface().(cbor.Er)
}

// ReturnObj returns a new instance of return object
func (mi *MethodInfo) ReturnObj() cbor.Er {
	if mi.Method.Ret == nil {
		return nil
	}

	return reflect.New(mi.Method.Ret.Elem()).Interface().(cbor.Er)
}

func (mi *MethodInfo) IsParamsImplemetsCbor() bool {
	if mi.Method.Params == nil {
		return true
	}

	return mi.Method.Params.Implements(reflect.TypeOf((*cbor.Er)(nil)).Elem())
}

func (mi *MethodInfo) IsRetImplemetsCbor() bool {
	if mi.Method.Ret == nil {
		return true
	}

	return mi.Method.Ret.Implements(reflect.TypeOf((*cbor.Er)(nil)).Elem())
}

func (mi MethodInfo) IsEmpty() bool {
	return mi == MethodInfo{}
}
