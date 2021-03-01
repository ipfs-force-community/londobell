package mcodec

import (
	"fmt"
	"reflect"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

var specialSchemaTypes = []reflect.Type{
	reflect.TypeOf([]byte(nil)),
}

var schemaReg = registry{
	codecs: map[reflect.Type]codec{},
}

// RegisterSchemaType register types for schema representation, usually interfaces
func RegisterSchemaType(tv interface{}, examples ...interface{}) {
	t := reflect.TypeOf(tv)
	isInterface := false
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Interface {
		t = t.Elem()
		isInterface = true
	}

	examVals := make([]reflect.Value, len(examples))
	for ei := range examples {
		ev := reflect.ValueOf(examples[ei])
		evt := ev.Type()
		if isInterface {
			if !evt.Implements(t) {
				panic(fmt.Errorf("#%d example of type %s does not implement interface %s", ei, evt, t))
			}
		} else {
			if evt != t {
				panic(fmt.Errorf("#%d example is %s, expecting %s", ei, evt, t))
			}
		}

		examVals[ei] = ev
	}

	schemaReg.Lock()
	schemaReg.codecs[t] = codec{
		t:        t,
		examples: examVals,
	}
	schemaReg.Unlock()
}

type schemaKindCodec struct {
}

func (s *schemaKindCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		pv := v
		if pv.IsNil() {
			pv = reflect.New(v.Type().Elem())
		}

		return bsoncodec.NewPointerCodec().EncodeValue(ctx, brw, pv)
	}

	return brw.WriteString(fmt.Sprintf("%s", v.Type()))
}

type schemaTypeCodec struct {
	t reflect.Type
}

func (s *schemaTypeCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	return brw.WriteString(fmt.Sprintf("%s", s.t))
}

func (s *schemaTypeCodec) DecodeValue(bsoncodec.DecodeContext, bsonrw.ValueReader, reflect.Value) error {
	return fmt.Errorf("schema codec should not be used for decoding")
}

// SchemaRegistry build a bsoncodec.Registry for output definitions
func SchemaRegistry() *bsoncodec.Registry {
	kinds := []reflect.Kind{
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Map,
		reflect.Slice,
		reflect.String,
		reflect.Ptr,
	}

	builder := bson.NewRegistryBuilder()

	// kinds
	kindCodec := &schemaKindCodec{}
	for _, k := range kinds {
		builder = builder.RegisterDefaultEncoder(k, kindCodec)
	}
	builder = builder.RegisterDefaultEncoder(reflect.Struct, structCodec)

	// interfaces
	schemaReg.RLock()
	for t := range schemaReg.codecs {
		t := t
		builder = builder.RegisterCodec(t, &schemaTypeCodec{t: t})
	}
	schemaReg.RUnlock()

	// custom types
	reg.RLock()
	for t := range reg.codecs {
		t := t
		builder = builder.RegisterCodec(t, &schemaTypeCodec{t: t})
	}
	reg.RUnlock()

	// other special types
	for ti := range specialSchemaTypes {
		t := specialSchemaTypes[ti]
		builder = builder.RegisterCodec(t, &schemaTypeCodec{t: t})
	}

	return builder.Build()
}

type schemeExampleCodec struct {
	inner codec
	count uint64
}

func (s *schemeExampleCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	exLen := len(s.inner.examples)
	if exLen == 0 {
		return s.inner.ValueEncoderFunc(ctx, brw, v)
	}

	next := atomic.AddUint64(&s.count, 1)
	return s.inner.ValueEncoderFunc(ctx, brw, s.inner.examples[int(next)%exLen])
}

func (s *schemeExampleCodec) DecodeValue(bsoncodec.DecodeContext, bsonrw.ValueReader, reflect.Value) error {
	return fmt.Errorf("example codec should not be used for decoding")
}

type schemaSliceExampleCodec struct{}

func (s schemaSliceExampleCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	enc := bsoncodec.NewSliceCodec()

	if !v.IsNil() && v.Len() > 0 {
		return enc.EncodeValue(ctx, brw, v)
	}

	return enc.EncodeValue(ctx, brw, reflect.MakeSlice(v.Type(), 1, 1))
}

type schemaPtrExampleCodec struct{}

func (s schemaPtrExampleCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	pv := v
	if pv.IsNil() {
		pv = reflect.New(v.Type().Elem())
	}

	return bsoncodec.NewPointerCodec().EncodeValue(ctx, brw, pv)
}

type schemaBytesExampleCodec struct{}

func (s schemaBytesExampleCodec) EncodeValue(ctx bsoncodec.EncodeContext, brw bsonrw.ValueWriter, v reflect.Value) error {
	pv := v
	if pv.Len() == 0 {
		pv = reflect.ValueOf([]byte("Hello"))
	}

	return bsoncodec.NewByteSliceCodec().EncodeValue(ctx, brw, pv)
}

// ExampleRegisry build a bsoncodec.Registry for output result based on pre-defined values for special types
func ExampleRegisry() *bsoncodec.Registry {
	builder := bson.NewRegistryBuilder()
	builder = builder.RegisterDefaultEncoder(reflect.Struct, structCodec)
	builder = builder.RegisterDefaultEncoder(reflect.Slice, schemaSliceExampleCodec{})
	builder = builder.RegisterDefaultEncoder(reflect.Ptr, schemaPtrExampleCodec{})
	builder = builder.RegisterEncoder(reflect.TypeOf([]byte(nil)), schemaBytesExampleCodec{})

	reg.RLock()
	for t := range reg.codecs {
		c := reg.codecs[t]
		builder = builder.RegisterCodec(c.t, &schemeExampleCodec{inner: c})
	}
	reg.RUnlock()

	return builder.Build()
}
