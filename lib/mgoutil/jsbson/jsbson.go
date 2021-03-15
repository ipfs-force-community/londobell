package jsbson

import (
	"fmt"

	"github.com/robertkrimen/otto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Object classes
const (
	Object   = "Object"
	Function = "Function"
	Array    = "Array"
	String   = "String"
	Number   = "Number"
	Boolean  = "Boolean"
	Date     = "Date"
	RegExp   = "RegExp"
)

// Parse generates a aggregation pipeline from the given source code with context
func Parse(ctx, src interface{}) (interface{}, error) {
	vm := otto.New()

	if ctx != nil {
		if err := vm.Set("ctx", ctx); err != nil {
			return nil, fmt.Errorf("set context: %w", err)
		}
	}

	v, err := vm.Eval(fmt.Sprintf("(%s)", src))
	if err != nil {
		return nil, fmt.Errorf("eval source: %w", err)
	}

	return value2agg(v)
}

func value2agg(v otto.Value) (interface{}, error) {
	if v.IsUndefined() {
		// {"$undefined": true}
		return primitive.Undefined{}, nil
	}

	if v.IsPrimitive() {
		return v.Export()
	}

	class := v.Class()
	switch class {
	case Function:
		// {"$code": string}
		return primitive.JavaScript(v.String()), nil

	case Object:
		return object2agg(v.Object())

	case Array:
		return array2agg(v.Object())

	case RegExp:
		return regexp2agg(v.Object())

	default:
		return nil, fmt.Errorf("unexpected class %s", class)
	}
}

func regexp2agg(obj *otto.Object) (interface{}, error) {
	srcField, err := obj.Get("source")
	if err != nil {
		return nil, err
	}

	source, err := srcField.ToString()
	if err != nil {
		return nil, err
	}

	flags := make([]byte, 0, 3)
	has, err := field2boolean(obj, "global")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'g')
	}

	has, err = field2boolean(obj, "ignoreCase")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'i')
	}

	has, err = field2boolean(obj, "multiline")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'm')
	}

	return primitive.Regex{
		Pattern: source,
		Options: string(flags),
	}, nil
}

func field2boolean(obj *otto.Object, field string) (bool, error) {
	v, err := obj.Get(field)
	if err != nil {
		return false, err
	}

	return v.ToBoolean()
}

func object2agg(obj *otto.Object) (interface{}, error) {
	keys := obj.Keys()
	out := make(bson.D, 0, len(keys))
	for _, key := range keys {
		field, err := obj.Get(key)
		if err != nil {
			return nil, fmt.Errorf("get object field %s: %w", key, err)
		}

		val, err := value2agg(field)
		if err != nil {
			return nil, fmt.Errorf("export value for object field %s: %w", key, err)
		}

		out = append(out, bson.E{
			Key:   key,
			Value: val,
		})
	}

	return out, nil
}

func array2agg(a *otto.Object) (interface{}, error) {
	keys := a.Keys()
	out := make([]interface{}, 0, len(keys))
	for i := range keys {
		ele, err := a.Get(keys[i])
		if err != nil {
			return nil, fmt.Errorf("get #%d element in array: %w", i, err)
		}

		val, err := value2agg(ele)
		if err != nil {
			return nil, fmt.Errorf("export value for #%d element in array: %w", i, err)
		}

		out = append(out, val)
	}

	return out, nil
}
