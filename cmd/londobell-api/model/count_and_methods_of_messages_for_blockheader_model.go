package model

type CountAndMethodsForBlockHeader struct {
	TotalCount int64    `bson:"totalCount" json:"totalCount"`
	AllMethods []string `bson:"methods"`
}
