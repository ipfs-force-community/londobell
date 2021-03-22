package mmetamgr

import (
	"context"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/mgoutil"
)

var log = logging.Logger("metamgr")

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

// Watch impl common.MetaManager
func (m *MetaMgr) Watch(ctx context.Context, key string, cb func(bson.RawValue) error) error {
	wlog := log.With("key", key)

	opt := options.ChangeStream().SetFullDocument(options.UpdateLookup)

	pipe := mongo.Pipeline{
		{
			{
				Key: "$match",
				Value: bson.D{
					{
						Key:   "documentKey._id",
						Value: key,
					},
					{
						Key: "operationType",
						Value: bson.D{
							{
								Key: "$in",
								Value: []string{
									"insert",
									"replace",
									"update",
								},
							},
						},
					},
				}},
		},
	}

	watch, err := m.col.Watch(ctx, pipe, opt)
	if err != nil {
		return fmt.Errorf("start watching: %w", err)
	}

	go func() {
		wlog.Info("watching change stream")
		defer wlog.Info("done watching change stream")

		for {
			select {
			case <-ctx.Done():
				return

			default:
			}

			if err := m.watch(ctx, watch, cb, wlog); err != nil {
				if !common.IsCtxCanceled(err) {
					wlog.Errorf("error occurs in change stream watching: %s", err)
				}
			}

			opt = opt.SetResumeAfter(watch.ResumeToken())
			select {
			case <-time.After(5 * time.Second):

			case <-ctx.Done():
				return
			}

			watch, err = m.col.Watch(ctx, pipe, opt)
			if err != nil {
				wlog.Errorf("failed to restart a change stream: %s", err)
				return
			}
		}
	}()

	return nil

}

type changeEvent struct {
	FullDocument struct {
		Detail bson.RawValue
	} `bson:"fullDocument"`

	DocumentKey struct {
		ID string `bson:"_id"`
	} `bson:"documentKey"`
}

func (m *MetaMgr) watch(ctx context.Context, w *mongo.ChangeStream, cb func(bson.RawValue) error, wl *zap.SugaredLogger) error {
	if w == nil {
		return fmt.Errorf("get nil change stream instance")
	}

	defer w.Close(ctx)

	for w.Next(ctx) {
		var event changeEvent
		err := w.Decode(&event)
		if err != nil {
			wl.Errorf("unable fo unmarshal change event: %s", err)
			continue
		}

		if err := cb(event.FullDocument.Detail); err != nil {
			wl.Errorf("error returned by event callback: %s", err)
		}
	}

	return w.Err()
}
