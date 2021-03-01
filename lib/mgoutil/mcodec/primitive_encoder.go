package mcodec

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

func encodeBytes(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.([]byte)
	if !ok {
		return fmt.Errorf("%w: want ([]byte), get %T", ErrInvalidValue, o)
	}

	return vw.WriteBinary(v)
}

func encodeBool(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.(bool)
	if !ok {
		return fmt.Errorf("%w: want (bool), get %T", ErrInvalidValue, o)
	}

	return vw.WriteBoolean(v)
}

func encodeFloat64(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.(float64)
	if !ok {
		return fmt.Errorf("%w: want (float64), get %T", ErrInvalidValue, o)
	}

	return vw.WriteDouble(v)
}

func encodeInt32(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.(int32)
	if !ok {
		return fmt.Errorf("%w: want (int32), get %T", ErrInvalidValue, o)
	}

	return vw.WriteInt32(v)
}

func encodeInt64(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.(int64)
	if !ok {
		return fmt.Errorf("%w: want (int64), get %T", ErrInvalidValue, o)
	}

	return vw.WriteInt64(v)
}

func encodeStr(vw bsonrw.ValueWriter, isNil bool, o interface{}) error {
	if isNil {
		return vw.WriteNull()
	}

	v, ok := o.(string)
	if !ok {
		return fmt.Errorf("%w: want (string), get %T", ErrInvalidValue, o)
	}

	return vw.WriteString(v)
}
