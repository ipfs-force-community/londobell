package mmetamgr

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dtynn/londobell/lib/mgoutil"
)

// New constructs a common.MetaManager
func New(initCtx context.Context, dsn string) (*MetaMgr, error) {
	ctx, cancel := context.WithTimeout(initCtx, 5*time.Second)
	defer cancel()

	client, err := mgoutil.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	db := client.Database("metamgr")
	col := db.Collection("meta")

	return &MetaMgr{
		cli: client,
		db:  db,
		col: col,

		upsertOpt: options.Update().SetUpsert(true),
	}, nil
}

type metaItem struct {
	Key    string `bson:"_id"`
	Detail bson.RawValue
}

// MetaMgr is an implementation of the common.MetaManager based on mongo
type MetaMgr struct {
	cli *mongo.Client
	db  *mongo.Database
	col *mongo.Collection

	upsertOpt *options.UpdateOptions
}

// Load impl common.MetaManager
func (m *MetaMgr) Load(ctx context.Context, key string, out interface{}) (bool, error) {
	var item metaItem

	err := m.col.FindOne(ctx, bson.M{"_id": key}).Decode(&item)
	if err != nil && err != mongo.ErrNoDocuments {
		return false, fmt.Errorf("load meta item: %w", err)
	}

	if err != nil {
		return false, nil
	}

	if err := item.Detail.Unmarshal(out); err != nil {
		return false, fmt.Errorf("unmarshal detail: %w", err)
	}

	return true, nil
}

// Update impl common.MetaManager
func (m *MetaMgr) Update(ctx context.Context, key string, val interface{}) error {
	_, err := m.col.UpdateOne(ctx, bson.M{"_id": key}, bson.M{"$set": bson.M{"Detail": val}}, m.upsertOpt)
	return err
}
