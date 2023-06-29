package common

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type Collections struct {
	DB   *mongo.Database
	Cols []*mongo.Collection
}
