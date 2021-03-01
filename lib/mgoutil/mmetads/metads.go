package mmetads

import (
	"context"
	"sync"
	"time"

	"github.com/ipfs/go-datastore"

	query "github.com/ipfs/go-datastore/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// common constants
const (
	DBName  = "metads"
	ColName = "meta"
)

var (
	_ datastore.Batching = (*mgoMetaStore)(nil)
	_ datastore.Batch    = (*mgoMetaBatch)(nil)
)

var (
	rootKey = datastore.RawKey("/")

	replaceOpt = options.Replace().SetUpsert(true)

	findSizeOpt = options.FindOne().SetProjection(bson.D{{Key: "s", Value: 1}})

	bulkWriteOpt = options.BulkWrite().SetOrdered(false)
)

// NewMgoDS returns a datastore.Batching based on mongo
func NewMgoDS(ctx context.Context, cli *mongo.Client, readOnly bool) (datastore.Batching, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &mgoMetaStore{
		runCtx:    ctx,
		runCancel: cancel,
		readOnly:  readOnly,
		cli:       cli,
		col:       cli.Database(DBName).Collection(ColName),
	}, nil
}

// MgoMetaItem is the document schema for mgo-metads
type MgoMetaItem struct {
	ID       string   `bson:"_id"`
	Prefixes []string `bson:"ps"`
	B        []byte   `bson:"b"`
	Size     int      `bson:"s"`
	Time     int64    `bson:"t"`
}

type mgoMetaStore struct {
	runCtx    context.Context
	runCancel context.CancelFunc

	readOnly bool

	cli *mongo.Client
	col *mongo.Collection
}

func (mms *mgoMetaStore) findOne(key datastore.Key, opts ...*options.FindOneOptions) (*MgoMetaItem, error) {
	var item MgoMetaItem
	err := mms.col.FindOne(mms.runCtx, findQuery(key), opts...).Decode(&item)
	if err == nil {
		return &item, nil
	}

	if err == mongo.ErrNoDocuments {
		return nil, datastore.ErrNotFound
	}

	return nil, err
}

func (mms *mgoMetaStore) Get(key datastore.Key) ([]byte, error) {
	item, err := mms.findOne(key)
	if err != nil {
		return nil, err
	}

	return item.B, nil
}

func (mms *mgoMetaStore) Has(key datastore.Key) (bool, error) {
	count, err := mms.col.CountDocuments(mms.runCtx, findQuery(key))
	return count != 0, err
}

func (mms *mgoMetaStore) GetSize(key datastore.Key) (int, error) {
	item, err := mms.findOne(key, findSizeOpt)
	if err != nil {
		return 0, err
	}

	return item.Size, err
}

func (mms *mgoMetaStore) Query(q query.Query) (query.Results, error) {
	filter := bson.M{}

	if prefix := datastore.NewKey(q.Prefix); prefix != rootKey {
		filter["ps"] = prefix.String()
	}

	findOpt := options.Find().SetCursorType(options.NonTailable)
	projection := bson.M{}

	if q.KeysOnly {
		projection["_id"] = 1

		if q.ReturnsSizes {
			projection["s"] = 1
		}
	}

	if len(projection) > 0 {
		findOpt = findOpt.SetProjection(projection)
	}

	if len(q.Orders) > 0 {
		switch q.Orders[0].(type) {
		case query.OrderByKey, *query.OrderByKey:
			findOpt = findOpt.SetSort(bson.M{"_id": 1})

		case query.OrderByKeyDescending, *query.OrderByKeyDescending:
			findOpt = findOpt.SetSort(bson.M{"_id": -1})

		default:
		}
	}

	cursor, err := mms.col.Find(mms.runCtx, filter, findOpt)
	if err != nil {
		return nil, err
	}

	return query.ResultsFromIterator(q, query.Iterator{
		Next: func() (query.Result, bool) {
			if !cursor.Next(mms.runCtx) {
				return query.Result{}, false
			}

			var item MgoMetaItem
			if err := cursor.Decode(&item); err != nil {
				return query.Result{
					Error: err,
				}, false
			}

			return query.Result{
				Entry: query.Entry{
					Key:   item.ID,
					Value: item.B,
					Size:  item.Size,
				},
			}, true

		},
		Close: func() error {
			return cursor.Close(mms.runCtx)
		},
	}), nil
}

func (mms *mgoMetaStore) Put(key datastore.Key, value []byte) error {
	if mms.readOnly {
		return nil
	}

	item, err := newMgoMetaItem(key, value)
	if err != nil {
		return err
	}

	_, err = mms.col.ReplaceOne(mms.runCtx, findQuery(key), item, replaceOpt)
	return err
}

func (mms *mgoMetaStore) Delete(key datastore.Key) error {
	if mms.readOnly {
		return nil
	}

	_, err := mms.col.DeleteOne(mms.runCtx, findQuery(key))
	return err
}

func (mms *mgoMetaStore) Sync(prefix datastore.Key) error {
	return nil
}

func (mms *mgoMetaStore) Close() error {
	mms.runCancel()
	return nil
}

func (mms *mgoMetaStore) Batch() (datastore.Batch, error) {
	return &mgoMetaBatch{
		ctx:      mms.runCtx,
		col:      mms.col,
		readOnly: mms.readOnly,
	}, nil
}

type mgoMetaBatch struct {
	ctx context.Context

	col *mongo.Collection

	readOnly bool

	bulkOnce sync.Once
	modelsMu sync.Mutex
	models   []mongo.WriteModel
}

func (mmb *mgoMetaBatch) Put(key datastore.Key, value []byte) error {
	if mmb.readOnly {
		return nil
	}

	mmb.modelsMu.Lock()
	defer mmb.modelsMu.Unlock()

	item, err := newMgoMetaItem(key, value)
	if err != nil {
		return err
	}

	mmb.models = append(mmb.models, mongo.NewReplaceOneModel().SetFilter(findQuery(key)).SetReplacement(item).SetUpsert(true))
	return nil
}

func (mmb *mgoMetaBatch) Delete(key datastore.Key) error {
	if mmb.readOnly {
		return nil
	}

	mmb.modelsMu.Lock()
	defer mmb.modelsMu.Unlock()

	mmb.models = append(mmb.models, mongo.NewDeleteOneModel().SetFilter(findQuery(key)))
	return nil
}

func (mmb *mgoMetaBatch) Commit() error {
	if len(mmb.models) == 0 {
		return nil
	}

	var err error
	mmb.bulkOnce.Do(func() {
		_, err = mmb.col.BulkWrite(mmb.ctx, mmb.models, bulkWriteOpt)
	})

	return err
}

func newMgoMetaItem(key datastore.Key, value []byte) (*MgoMetaItem, error) {
	id, prefixes, err := extractKey(key)
	if err != nil {
		return nil, err
	}

	return &MgoMetaItem{
		ID:       id,
		Prefixes: prefixes,
		B:        value,
		Size:     len(value),
		Time:     time.Now().Unix(),
	}, nil
}

func findQuery(key datastore.Key) bson.D {
	return bson.D{{Key: "_id", Value: key.String()}}
}

func extractKey(key datastore.Key) (string, []string, error) {
	prefixes := make([]string, 0, 8)

	parent := key.Parent()
	for parent != rootKey {
		prefixes = append(prefixes, parent.String())
		parent = parent.Parent()
	}

	return key.String(), prefixes, nil
}
