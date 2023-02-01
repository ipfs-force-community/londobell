package model

type MinersForOwnerRes struct {
	Owner  string `bson:"_id" json:"_id"`
	Miners []string
}
