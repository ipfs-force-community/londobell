package model

type AllMethodsRes struct {
	//Addrs      AddrList `bson:"_id"`
	AllMethods []string `bson:"all_methods" json:"allMethods"`
}

type AddrList struct {
	From string `bson:"from"`
	To   string `bson:"to"`
}
