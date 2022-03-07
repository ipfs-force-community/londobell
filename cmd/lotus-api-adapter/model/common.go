package model

type State = uint64

const (
	Success State = iota
	Fail
)

type CommonRes struct {
	Code uint64      `json:"code" example:"1"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
