package model

type Count struct {
	Count int64 `json:"Count" bson:"count"`
}

type CountRes struct {
	Count int64
}

type ByMethodNameCountRes struct {
	MethodName string `bson:"_id" json:"_id"`
	Count      int64
}
