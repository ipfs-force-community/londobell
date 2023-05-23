package model

type AllMethodsRes struct {
	MethodName string
	Count      int64
}

type AllMethodsForActorRes struct {
	MethodName string `bson:"_id" json:"_id"`
	Count      int64
}

type AllActorsMethodsRes struct {
	ActorIDMethod ActorIDMethodRes `bson:"_id" json:"_id"`
	Count         int64
}

type ActorIDMethodRes struct {
	ActorID    string
	MethodName string
}

type AllActorsMsgsCountRes struct {
	ActorID string `bson:"_id" json:"_id"`
	Count   int64
}

type AllBlockMethodNamesRes struct {
	MethodNames []string
}
