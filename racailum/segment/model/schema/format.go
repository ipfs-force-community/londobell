package schema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"

	"github.com/ipfs-force-community/londobell/common"
)

// FormatJSON tries to marshal object into json, with bson codec
func FormatJSON(r *bsoncodec.Registry, v common.Document, withIndent bool) ([]byte, error) {
	var buf bytes.Buffer
	jrw, err := bsonrw.NewExtJSONValueWriter(&buf, false, false)
	if err != nil {
		return nil, err
	}

	encoder, err := bson.NewEncoderWithContext(bsoncodec.EncodeContext{
		Registry: r,
	}, jrw)
	if err != nil {
		return nil, err
	}

	err = encoder.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("marshal ext json: %w", err)
	}

	b := buf.Bytes()

	if !withIndent {
		return b, nil
	}

	var jv interface{}
	err = json.Unmarshal(b, &jv)
	if err != nil {
		return nil, fmt.Errorf("unmarshal as json: %w", err)
	}

	return json.MarshalIndent(jv, "", "\t")
}
