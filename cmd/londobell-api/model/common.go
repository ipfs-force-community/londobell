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
	Addrs      []string `json:"addrs"`
	Method     uint64   `json:"method"`
	MethodName string   `json:"method_name"`
	Cid        string   `json:"cid"`
	Cids       []string `json:"cids"`
	ID         uint64   `json:"id"`
	Sort       int      `json:"sort"`
	To         string   `json:"to"`
	Index      int64    `json:"index"`
	Limit      int64    `json:"limit"`
}

type Ctx struct {
	StartEpoch int64
	EndEpoch   int64
	Addr       string
	PrimaryID  string
	Addrs      []string
	Method     uint64
	MethodName string
	Cid        string
	Cids       []string
	ID         uint64
	IDStr      string
	Sort       int
	To         string
	Skip       int64
	Limit      int64
	ProviderID string
	ClientID   string
}
