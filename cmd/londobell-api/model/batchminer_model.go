package model

type Miner struct {
	Miner string `json:"miner"`
}

type BatchMinersReq struct {
	Miners []Miner `json:"miners"`
	Epoch  int64   `json:"epoch"`
}

type BatchMinersRes struct {
	MinersRes []MinerRes `json:"minersRes"`
}
