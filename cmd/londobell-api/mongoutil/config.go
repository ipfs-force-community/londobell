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
	TipSetCol            *mongo.Collection
	ActorStateCol        *mongo.Collection
	MinerFundsCol        *mongo.Collection
	BlockHeaderCol       *mongo.Collection
	ClaimedPowerCol      *mongo.Collection
	DealProposalCol      *mongo.Collection
	MessageCol           *mongo.Collection
	MessageBlockCol      *mongo.Collection

	DbConfig = &DBConfig{}

	TmpClient          *mongo.Client
	TmpTraceCol        *mongo.Collection
	TmpTipSetCol       *mongo.Collection
	TmpBlockHeaderCol  *mongo.Collection
	TmpFinalHeightCol  *mongo.Collection
	TmpMessageCol      *mongo.Collection
	TmpMessageBlockCol *mongo.Collection
)

type DBConfig struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	TmpURL  string `json:"tmp-url"`
	TmpName string `json:"tmp-name"`
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
