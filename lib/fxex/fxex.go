package fxex

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// As generates an fx constructor to use value as given type
func As(val, as interface{}) interface{} {
	rval := reflect.ValueOf(val)
	rtype := rval.Type()

	asType := reflect.ValueOf(as).Type().Elem()
	if !rtype.ConvertibleTo(asType) {
		panic(fmt.Errorf("value of type %s is not convertible to %s", rtype, asType))
	}

	funcType := reflect.FuncOf([]reflect.Type{}, []reflect.Type{asType}, false)
	return reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{rval.Convert(asType)}
	}).Interface()
}

// Convert generates an fx constructor for from => to
func Convert(from, to interface{}) interface{} {
	fromType := reflect.ValueOf(from).Type().Elem()
	toType := reflect.ValueOf(to).Type().Elem()
	if !fromType.ConvertibleTo(toType) {
		panic(fmt.Errorf("from type %s is not convertible to %s", fromType, toType))
	}

	ctorType := reflect.FuncOf([]reflect.Type{fromType}, []reflect.Type{toType}, false)
	return reflect.MakeFunc(ctorType, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{args[0].Convert(toType)}
	}).Interface()
}

// ProvideEx automatically makes constructor fon non-func objects, and generates fx.Option from the given constructors
func ProvideEx(constructors ...interface{}) fx.Option {
	ctors := make([]interface{}, len(constructors))
	for i := range constructors {
		ctor := constructors[i]

		rval := reflect.ValueOf(ctor)
		rtyp := rval.Type()
		rkind := rtyp.Kind()
		if rkind == reflect.Func {
			ctors[i] = ctor
			continue
		}

		funcType := reflect.FuncOf([]reflect.Type{}, []reflect.Type{rtyp}, false)
		ctors[i] = reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
			return []reflect.Value{rval}
		}).Interface()
	}

	return fx.Provide(ctors...)
}
