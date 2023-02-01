package model

type AllOwnersRes struct {
	Count     int      `bson:"count"`
	AllOwners []string `bson:"allOwners"`
}
