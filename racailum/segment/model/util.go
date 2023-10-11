package model

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"go.mongodb.org/mongo-driver/bson"
)

func rangedFilter(field string, lower, upper *abi.ChainEpoch) bson.M {
	inner := bson.M{}
	if lower != nil {
		inner["$gt"] = *lower
	}

	if upper != nil {
		inner["$lt"] = *upper
	}

	return bson.M{field: inner}
}

func getColName(doc interface{}) string {
	return reflect.TypeOf(doc).Name()
}

func extractEpochFieldName(doc interface{}) string {
	field, ok := reflect.TypeOf(doc).FieldByName("Epoch")
	if !ok {
		panic(fmt.Errorf("Epoch field not found for %T", doc))
	}

	if tagstr := field.Tag.Get("bson"); tagstr != "" {
		if name := strings.Split(tagstr, ",")[0]; name != "" {
			return name
		}
	}

	return field.Name
}

// 获取父消息在mongo中的_id
func GetRootID(child string) (string, error) {
	subs := strings.Split(child, "-")
	if len(subs) < 2 {
		return "", fmt.Errorf("getRootids Split length err: %s", child)
	}
	rootID := subs[0] + "-" + subs[1]
	return rootID, nil
}
