package multiquery

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type Collections struct {
	DB   *mongo.Database
	Cols []*mongo.Collection
}

func NewDBCollections() DBCollections {
	return DBCollections{
		DBCollectionsMap: make(map[string]Collections),
	}
}
