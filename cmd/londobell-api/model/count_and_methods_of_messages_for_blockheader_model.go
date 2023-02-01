package model

type CountAndMethodsForBlockHeader struct {
	MethodName string `bson:"_id" json:"_id"`
	Count      int64
}
