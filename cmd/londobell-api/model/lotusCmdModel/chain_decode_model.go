package lotusCmdModel

import cbg "github.com/whyrusleeping/cbor-gen"

type ChainDecodeReq struct {
	To       string `json:"to"`
	Method   string `json:"method"`
	Params   string `json:"params"`
	Epoch    int64  `json:"epoch"`
	Encoding string `json:"encoding" default:"base64"`
}

type ChainDecodeRes struct {
	Params cbg.CBORUnmarshaler `json:"params"`
}
