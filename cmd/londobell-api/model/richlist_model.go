package model

type RichListRes struct {
	TotalCount int64 `json:"totalCount"`
	RichList   []Rich
}

type Rich struct {
	Actor   string `bson:"Addr"`
	Balance string `bson:"Balance"`
	Type    string `bson:"Code"`
}
