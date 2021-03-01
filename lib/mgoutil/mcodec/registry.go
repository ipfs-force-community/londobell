package mcodec

import (
	"fmt"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

// common errors
var (
	ErrFuncRequired       = fmt.Errorf("func required")
	ErrInvalidEncoderFunc = fmt.Errorf("invalid encoder func")
	ErrInvalidDecoderFunc = fmt.Errorf("invalid decoder func")
	ErrInvalidValue       = fmt.Errorf("invalid value")
	ErrDuplicateCodec     = fmt.Errorf("duplicate codec")
)

var (
	_ bsoncodec.ValueCodec = (*codec)(nil)
)

type codec struct {
	t        reflect.Type
	examples []reflect.Value
	bsoncodec.ValueEncoderFunc
	bsoncodec.ValueDecoderFunc
}

var reg = registry{
	codecs: map[reflect.Type]codec{},
}

type registry struct {
	sync.RWMutex
	codecs map[reflect.Type]codec
}

func (r *registry) registerCodec(t reflect.Type, efn bsoncodec.ValueEncoderFunc, dfn bsoncodec.ValueDecoderFunc, examples []reflect.Value) bool {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.codecs[t]; ok {
		return false
	}

	r.codecs[t] = codec{
		t:                t,
		examples:         examples,
		ValueEncoderFunc: efn,
		ValueDecoderFunc: dfn,
	}

	return true
}

var setupOnce sync.Once

// Setup finalize the registration of codecs and set them into bson.DefaultRegistry
func Setup() {
	setupOnce.Do(func() {
		builder := bson.NewRegistryBuilder()
		builder = builder.RegisterDefaultDecoder(reflect.Struct, structCodec)
		builder = builder.RegisterDefaultEncoder(reflect.Struct, structCodec)

		reg.RLock()
		defer reg.RUnlock()

		for t, c := range reg.codecs {
			builder = builder.RegisterCodec(t, c)
		}

		bson.DefaultRegistry = builder.Build()
	})
}

// RegisterCodec registers encode funcs like func(T) (primitive, bool, error) and decode funcs like func(primitive) (T, error)
func RegisterCodec(enc interface{}, dec interface{}, allowPtr bool, examples ...interface{}) error {
	encv := reflect.ValueOf(enc)
	enck := encv.Kind()
	if enck != reflect.Func {
		return fmt.Errorf("%w: value is a %s", ErrInvalidEncoderFunc, enck)
	}

	enct := encv.Type()
	{
		if numIn := enct.NumIn(); numIn != 1 {
			return fmt.Errorf("%w: got %d in", ErrInvalidEncoderFunc, numIn)
		}

		if numOut := enct.NumOut(); numOut != 3 {
			return fmt.Errorf("%w: got %d out", ErrInvalidEncoderFunc, numOut)
		}

		if eout1 := enct.Out(1); eout1 != boolType {
			return fmt.Errorf("%w: 2nd out should be bool, got %s", ErrInvalidEncoderFunc, eout1)
		}

		if eout2 := enct.Out(2); eout2 != errorType {
			return fmt.Errorf("%w: 3rd out should be error, got %s", ErrInvalidEncoderFunc, eout2)
		}
	}

	einType := enct.In(0)
	if _, ok := primitiveCodecs[einType]; ok {
		return fmt.Errorf("%w: input should not be primitive, got %s", ErrInvalidEncoderFunc, einType)
	}

	exampleVals := make([]reflect.Value, len(examples))
	for ei := range examples {
		examVal := reflect.ValueOf(examples[ei])
		if examTyp := examVal.Type(); examTyp != einType {
			return fmt.Errorf("got #%d example value of unexpected type %s, want %s", ei, examTyp, einType)
		}

		exampleVals[ei] = examVal
	}

	eout0 := enct.Out(0)
	primitive, ok := primitiveCodecs[eout0]
	if !ok {
		return fmt.Errorf("%w: output type %s is not primitive", ErrInvalidEncoderFunc, eout0)
	}

	decv := reflect.ValueOf(dec)
	deck := decv.Kind()
	if deck != reflect.Func {
		return fmt.Errorf("%w: value is a %s", ErrInvalidDecoderFunc, enck)
	}

	dect := decv.Type()
	{

		if numIn := dect.NumIn(); numIn != 1 {
			return fmt.Errorf("%w: got %d in", ErrInvalidDecoderFunc, numIn)
		}

		if numOut := dect.NumOut(); numOut != 2 {
			return fmt.Errorf("%w: got %d out", ErrInvalidDecoderFunc, numOut)
		}

		if din0 := dect.In(0); din0 != eout0 {
			return fmt.Errorf("%w: 1st in should be %s, got %s", ErrInvalidDecoderFunc, eout0, din0)
		}

		if dout0 := dect.Out(0); dout0 != einType {
			return fmt.Errorf("%w: 1st in should be %s, got %s", ErrInvalidDecoderFunc, einType, dout0)
		}

		if dout1 := dect.Out(1); dout1 != errorType {
			return fmt.Errorf("%w: 2nd out should be error, got %s", ErrInvalidDecoderFunc, dout1)
		}

	}

	// for inType
	done := reg.registerCodec(einType,
		func(ctx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, rv reflect.Value) error {
			outputs := encv.Call([]reflect.Value{rv})
			if !outputs[2].IsNil() {
				return outputs[2].Interface().(error)
			}

			useNil := outputs[1].Interface().(bool)

			return primitive.e(vw, useNil, outputs[0].Interface())
		},

		func(ctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, rv reflect.Value) error {
			if !rv.CanSet() {
				return fmt.Errorf("%w: not changeable", ErrInvalidValue)
			}

			pv, isNil, err := primitive.d(vr)
			if err != nil {
				return err
			}

			if isNil {
				return nil
			}

			outputs := decv.Call([]reflect.Value{reflect.ValueOf(pv)})
			if !outputs[1].IsNil() {
				return outputs[1].Interface().(error)
			}

			rv.Set(outputs[0])
			return nil
		},

		exampleVals,
	)

	if !done {
		return fmt.Errorf("%w: for %s", ErrDuplicateCodec, einType)
	}

	if allowPtr && einType.Kind() != reflect.Ptr {
		einPtrType := reflect.New(einType).Type()

		ptrExampleVals := make([]reflect.Value, len(exampleVals))
		for ei := range exampleVals {
			ptrExamVal := reflect.New(einType)
			ptrExamVal.Elem().Set(exampleVals[ei])
			ptrExampleVals[ei] = ptrExamVal
		}

		done = reg.registerCodec(
			einPtrType,
			func(ctx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, rv reflect.Value) error {
				if rv.IsNil() {
					return vw.WriteNull()
				}

				outputs := encv.Call([]reflect.Value{rv.Elem()})
				if !outputs[2].IsNil() {
					return outputs[2].Interface().(error)
				}

				useNil := outputs[1].Interface().(bool)

				return primitive.e(vw, useNil, outputs[0].Interface())
			},

			func(ctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, rv reflect.Value) error {
				if !rv.CanSet() {
					return fmt.Errorf("%w: not changeable", ErrInvalidValue)
				}

				pv, isNil, err := primitive.d(vr)
				if err != nil {
					return err
				}

				if isNil {
					return nil
				}

				outputs := decv.Call([]reflect.Value{reflect.ValueOf(pv)})
				if !outputs[1].IsNil() {
					return outputs[1].Interface().(error)
				}

				newValue := reflect.New(einType)
				newValue.Elem().Set(outputs[0])

				rv.Set(newValue.Elem())
				return nil
			},

			ptrExampleVals,
		)

		if !done {
			return fmt.Errorf("%w: for %s", ErrDuplicateCodec, einPtrType)
		}
	}

	return nil
}
