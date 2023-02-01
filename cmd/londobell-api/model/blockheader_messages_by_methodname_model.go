package model

type BlockHeaderMessagesByMethodNameRes struct {
	Messages   []TraceForMessageRes
	TotalCount int64 `bson:"totalCount"`
}
