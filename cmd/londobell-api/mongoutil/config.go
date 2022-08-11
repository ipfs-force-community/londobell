package mongoutil

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	Client               *mongo.Client
	TraceCol             *mongo.Collection
	ActorBalanceCol      *mongo.Collection
	FinalHeightCol       *mongo.Collection
	MinerSectorHealthCol *mongo.Collection

	DbConfig *DBConfig
)

type DBConfig struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

func InitDB() {
	file, err := os.Open("./cmd/londobell-api/mongoutil/config.json")
	defer file.Close() //nolint:staticcheck
	if err != nil {
		panic(err)
	}

	configByte, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	DbConfig = &DBConfig{}
	err = json.Unmarshal(configByte, DbConfig)
	if err != nil {
		panic(err)
	}
}
