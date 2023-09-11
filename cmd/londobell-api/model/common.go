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
	StartEpoch           int64    `json:"start"`
	EndEpoch             int64    `json:"end"`
	Addr                 string   `json:"addr"`
	Addrs                []string `json:"addrs"`
	Method               uint64   `json:"method"`
	MethodName           string   `json:"method_name"`
	Cid                  string   `json:"cid"`
	Cids                 []string `json:"cids"`
	ID                   uint64   `json:"id"`
	Sort                 int      `json:"sort"`
	To                   string   `json:"to"`
	Index                int64    `json:"index"`
	Limit                int64    `json:"limit"`
	Hash                 string   `json:"hash"`
	TransferType         string   `json:"transfer-type"`
	CurEpoch             int64    `json:"cur_epoch"`
	ExpirationStartEpoch int64    `json:"expiration_start_epoch"`
	ExpirationEndEpoch   int64    `json:"expiration_end_epoch"`
	SectorNumber         int64    `json:"sector_number"`
	SectorSize           uint64   `json:"sector_size"`
}

type Ctx struct {
	StartEpoch           int64
	EndEpoch             int64
	Addr                 string
	PrimaryID            string
	Addrs                []string
	Method               uint64
	MethodName           string
	Cid                  string
	Cids                 []string
	ID                   uint64
	IDStr                string
	Sort                 int
	To                   string
	Skip                 int64
	Limit                int64
	Start                int64
	End                  int64
	TransferType         string
	CurEpoch             int64
	ExpirationStartEpoch int64
	ExpirationEndEpoch   int64
	SectorNumber         int64
	SectorSize           uint64
}
