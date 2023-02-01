package model

type RichListRes struct {
	TotalCount int64 `json:"totalCount"`
	RichList   []Rich
}

type Rich struct {
	Addr    string
	Balance string
	Code    string
}
