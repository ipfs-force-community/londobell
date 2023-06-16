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
	StartEpoch int64  `json:"start"`
	EndEpoch   int64  `json:"end"`
	Addr       string `json:"addr"`
}

type Ctx struct {
	StartEpoch int64
	EndEpoch   int64
	Addr       string
	ID         string
}
