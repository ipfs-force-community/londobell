package model

type AllActorsRes struct {
	AllFroms []string `bson:"all_froms"`
	AllTos   []string `bson:"all_tos"`
}
