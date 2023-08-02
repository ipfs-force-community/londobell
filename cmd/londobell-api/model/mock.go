package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	//log            = logging.Logger("mock")
	MockDecimal128 primitive.Decimal128
)

func init() {
	val, err := primitive.ParseDecimal128("123")
	if err != nil {
		panic(err)
	}

	MockDecimal128 = val
}
