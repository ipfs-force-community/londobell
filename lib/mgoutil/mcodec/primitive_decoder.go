package mcodec

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// common errors
var (
	ErrInvalidBSONTypeForDecoding = fmt.Errorf("invalid bson type for decoding")
)

func decodeBytes(vr bsonrw.ValueReader) ([]byte, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return nil, true, vr.ReadNull()

	case bsontype.Binary:
		b, _, err := vr.ReadBinary()
		return b, false, err

	default:
		return nil, false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.Binary, rt)
	}
}

func decodeBool(vr bsonrw.ValueReader) (bool, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return false, true, vr.ReadNull()

	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		return b, false, err

	default:
		return false, false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.Boolean, rt)
	}
}

func decodeFloat64(vr bsonrw.ValueReader) (float64, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return 0, true, vr.ReadNull()

	case bsontype.Double:
		b, err := vr.ReadDouble()
		return b, false, err

	default:
		return 0, false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.Double, rt)
	}
}

func decodeInt32(vr bsonrw.ValueReader) (int32, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return 0, true, vr.ReadNull()

	case bsontype.Int32:
		b, err := vr.ReadInt32()
		return b, false, err

	default:
		return 0, false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.Int32, rt)
	}
}

func decodeInt64(vr bsonrw.ValueReader) (int64, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return 0, true, vr.ReadNull()

	case bsontype.Int64:
		b, err := vr.ReadInt64()
		return b, false, err

	default:
		return 0, false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.Int64, rt)
	}
}

func decodeStr(vr bsonrw.ValueReader) (string, bool, error) {
	rt := vr.Type()
	switch rt {
	case bsontype.Null:
		return "", true, vr.ReadNull()

	case bsontype.String:
		b, err := vr.ReadString()
		return b, false, err

	default:
		return "", false, fmt.Errorf("%w: want %s, got %s", ErrInvalidBSONTypeForDecoding, bsontype.String, rt)
	}
}
