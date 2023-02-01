package model

type CountOfBlockMessageByMethodName struct {
	MethodName string `bson:"_id"`
	Count      int64  `bson:"count"`
}
