package model

//type CountOfActorMessagesByMethodName struct {
//	MethodName string   `bson:"_id"`
//	AllFroms   []string `bson:"all_froms"`
//	AllTos     []string `bson:"all_tos"`
//}

type CountOfActorMessagesByMethodName struct {
	Method string
	From   string
	To     string
}
