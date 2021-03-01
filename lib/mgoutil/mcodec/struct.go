package mcodec

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
)

func init() {
	codec, err := bsoncodec.NewStructCodec(defaultStructTagParser)
	if err != nil {
		panic(fmt.Errorf("construct struct codec: %w", err))
	}

	structCodec = codec
}

var structCodec *bsoncodec.StructCodec

// from https://github.com/mongodb/mongo-go-driver/blob/0f54b6155e6fa4aab38ee12979c7aba60f8dbdf7/bson/bsoncodec/struct_tag_parser.go#L88-L123
// but do not convert field name to lowercase
var defaultStructTagParser bsoncodec.StructTagParserFunc = func(sf reflect.StructField) (bsoncodec.StructTags, error) {
	key := sf.Name
	tag, ok := sf.Tag.Lookup("bson")
	if !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0 {
		tag = string(sf.Tag)
	}
	return parseTags(key, tag)
}

func parseTags(key string, tag string) (bsoncodec.StructTags, error) {
	var st bsoncodec.StructTags
	if tag == "-" {
		st.Skip = true
		return st, nil
	}

	for idx, str := range strings.Split(tag, ",") {
		if idx == 0 && str != "" {
			key = str
		}
		switch str {
		case "omitempty":
			st.OmitEmpty = true
		case "minsize":
			st.MinSize = true
		case "truncate":
			st.Truncate = true
		case "inline":
			st.Inline = true
		}
	}

	st.Name = key

	return st, nil
}
