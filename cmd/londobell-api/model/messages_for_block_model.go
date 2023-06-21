package model

import "encoding/json"

type MessagesForBlock struct {
	TotalCount int64    `bson:"totalCount" json:"totalCount"`
	Messages   []string `bson:"messages" json:"Messages"`
}

func (mb *MessagesForBlock) PrintRes() (string, error) {
	b, err := json.Marshal(mb)
	if err != nil {
		return "", err
	}

	return string(b), err
}
