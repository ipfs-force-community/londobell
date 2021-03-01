package mcodec

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

type encodingFunc = func(vw bsonrw.ValueWriter, useNil bool, v interface{}) error
type decodingFunc = func(vr bsonrw.ValueReader) (interface{}, bool, error)

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()

	bytesType   = reflect.TypeOf((*[]byte)(nil)).Elem()
	boolType    = reflect.TypeOf((*bool)(nil)).Elem()
	float64Type = reflect.TypeOf((*float64)(nil)).Elem()
	int32Type   = reflect.TypeOf((*int32)(nil)).Elem()
	int64Type   = reflect.TypeOf((*int64)(nil)).Elem()
	strType     = reflect.TypeOf((*string)(nil)).Elem()
)

type primitiveCodec struct {
	zero reflect.Value
	e    encodingFunc
	d    decodingFunc
}

var primitiveCodecs = map[reflect.Type]primitiveCodec{
	bytesType: primitiveCodec{
		zero: reflect.ValueOf([]byte{}),
		e:    encodeBytes,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeBytes(vr)
			return v, useNil, err
		},
	},

	boolType: primitiveCodec{
		zero: reflect.ValueOf(false),
		e:    encodeBool,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeBool(vr)
			return v, useNil, err
		},
	},

	float64Type: primitiveCodec{
		zero: reflect.ValueOf(float64(0)),
		e:    encodeFloat64,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeFloat64(vr)
			return v, useNil, err
		},
	},

	int32Type: primitiveCodec{
		zero: reflect.ValueOf(int32(0)),
		e:    encodeInt32,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeInt32(vr)
			return v, useNil, err
		},
	},

	int64Type: primitiveCodec{
		zero: reflect.ValueOf(int64(0)),
		e:    encodeInt64,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeInt64(vr)
			return v, useNil, err
		},
	},

	strType: primitiveCodec{
		zero: reflect.ValueOf(""),
		e:    encodeStr,
		d: func(vr bsonrw.ValueReader) (interface{}, bool, error) {
			v, useNil, err := decodeStr(vr)
			return v, useNil, err
		},
	},
}
