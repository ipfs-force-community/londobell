package lotusCmdModel

type ChainGetReq struct {
	Epoch  int64  `json:"epoch"`
	Path   string `json:"path"`
	AsType string `json:"as_type"`
}

type ChainGetRes struct {
	Path    string      `json:"path"`
	DAGNode interface{} `json:"dag_node"`
}
