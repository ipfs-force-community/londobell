package model

type AllMethodsForFromActor struct {
	FromActor  string   `bson:"_id" json:"FromActor"`
	AllMethods []string `bson:"all_methods"`
}

type AllMethodsForToActor struct {
	ToActor    string   `bson:"_id"`
	AllMethods []string `bson:"all_methods"`
}

type AllMethodsForActor struct {
	Actor      string
	AllMethods []string
}
