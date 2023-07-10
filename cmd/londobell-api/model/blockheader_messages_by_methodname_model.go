package model

type BlockHeaderMessagesByMethodNameRes struct {
	Messages   []BlockHeaderMessage
	TotalCount int64
}

type CountOfBlockHeaderMessagesByMethodNameRes struct {
	Epoch      int64 `json:"_id"`
	TotalCount int64 `json:"totalCount"`
}

type BlockHeaderMessage struct {
	Cid      string
	Epoch    int64
	Value    string
	From     string
	To       string
	ExitCode int64
	Method   string
}
