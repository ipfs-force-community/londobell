package model

type State = uint64

const (
	Success State = iota
	Fail
	NotFound
)

type CommonRes struct {
	Code uint64      `json:"code" example:"1"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type CommonReq struct {
	StartEpoch int64    `json:"start"`
	EndEpoch   int64    `json:"end"`
	Addr       string   `json:"addr"`
	Sort       int      `json:"sort"`
	To         string   `json:"to"`
	Method     uint64   `json:"method"`
	Count      uint64   `json:"count"`
	ID         uint64   `json:"id"`
	Cid        string   `json:"cid"`
	Cids       []string `json:"cids"`
	Index      int64    `json:"index"`
	Limit      int64    `json:"limit"` // int64
	MethodName string   `json:"method_name"`
}

type Ctx struct {
	StartEpoch int64
	EndEpoch   int64
	Addr       string
	Sort       int
	To         string
	Method     uint64
	Count      uint64
	ID         uint64
	Cid        string
	Cids       []string
	Skip       int64
	Limit      int64
	MethodName string
}
