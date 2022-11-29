package lotusCmdModel

type ChainEncodeReq struct {
	//Epoch     int64  `json:"epoch"`
	Method    string `json:"method"`
	To        string `json:"to"`
	Params    string `json:"params"`
	Encodeing string `json:"encoding" default:"base64"`
}

type ChainEncodeRes struct {
	Params string `json:"params"`
}
