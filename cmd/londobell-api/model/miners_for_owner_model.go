package model

type MinersForOwnerRes struct {
	Owner  string   `bson:"_id" json:"Owner"`
	Miners []string `bson:"Addrs" json:"Miners"`
}
