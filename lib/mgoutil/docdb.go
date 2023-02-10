package mgoutil

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"
)

var log = logging.Logger("mgoutil")

// NewMgoDocDB returns an instance of MgoDocDB
func NewMgoDocDB(ctx context.Context, cli *mongo.Client, db *mongo.Database) (*MgoDocDB, error) {
	mdb := &MgoDocDB{
		cli:       cli,
		db:        db,
		insertOpt: options.InsertMany().SetOrdered(false),
		updateOpt: options.Update().SetUpsert(true),
		aggOpt:    options.Aggregate(),
	}

	mdb.cols.m = make(map[string]*mongo.Collection)
	return mdb, nil
}

// MgoDocDB is an implementation of the common.DocumentDB, based on mongo
type MgoDocDB struct {
	cli *mongo.Client
	db  *mongo.Database

	insertOpt *options.InsertManyOptions
	updateOpt *options.UpdateOptions
	aggOpt    *options.AggregateOptions

	cols struct {
		sync.RWMutex
		m map[string]*mongo.Collection
	}
}

// Insert impl common.DocumentDB
func (m *MgoDocDB) Insert(ctx context.Context, colName string, docs []interface{}) (int, error) {
	res, err := m.getCol(colName).InsertMany(ctx, docs, m.insertOpt)
	var inserted int
	if res != nil {
		inserted = len(res.InsertedIDs)
	}

	if err != nil {
		if actualErr := extractActualMgoErrors(err); actualErr != nil {
			return inserted, actualErr
		}
	}

	return inserted, nil
}

func (m *MgoDocDB) Find(ctx context.Context, colName string, filter interface{},
	opts ...*options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := m.getCol(colName).Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (m *MgoDocDB) Update(ctx context.Context, colName string, filter, docs interface{}) (int, error) {
	res, err := m.getCol(colName).UpdateMany(ctx, filter, docs, m.updateOpt)
	var upserted int
	if res != nil {
		upserted = int(res.UpsertedCount) // todo: more log, such as UpsertedID
	}

	return upserted, err
}

// Delete impl common.DocumentDB
func (m *MgoDocDB) Delete(ctx context.Context, colName string, filter interface{}) (int, error) {
	res, err := m.getCol(colName).DeleteMany(ctx, filter)
	var deleted int
	if res != nil {
		deleted = int(res.DeletedCount)
	}

	return deleted, err
}

// Aggregate impl common.DocumentDB
func (m *MgoDocDB) Aggregate(ctx context.Context, colName string, pipeline interface{}, res interface{}) error {
	cur, err := m.getCol(colName).Aggregate(ctx, pipeline, m.aggOpt)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)
	if res != nil {
		if err := cur.All(ctx, res); err != nil {
			return err
		}
	}

	return nil
}

func (m *MgoDocDB) getCol(name string) *mongo.Collection {
	m.cols.RLock()
	col, ok := m.cols.m[name]
	m.cols.RUnlock()

	if !ok {
		m.cols.Lock()
		col, ok = m.cols.m[name]
		if !ok {
			col = m.db.Collection(name)
			m.cols.m[name] = col
		}
		m.cols.Unlock()
	}
	return col
}

func extractActualMgoErrors(err error) error {
	mbwr, ok := err.(mongo.BulkWriteException)
	if !ok {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}

		return err
	}

	var merr error
	for _, we := range mbwr.WriteErrors {
		// from mongo.IsDuplicateKeyError
		if we.Code == 11000 || we.Code == 11001 || we.Code == 12582 {
			continue
		}

		if we.Code == 16460 && strings.Contains(we.Message, " E11000 ") {
			continue
		}

		merr = multierror.Append(merr, err)
	}

	return merr
}

// multi write dbs
type MultiDB struct {
	dbs []*MgoDocDB
}

func (m *MultiDB) Insert(ctx context.Context, col string, docs []interface{}) (int, error) {
	var (
		inserted int
		err      error
	)

	for _, db := range m.dbs {
		inserted, err = db.Insert(ctx, col, docs)
		if err != nil {
			return inserted, err
		}
	}

	return inserted, nil
}

func (m *MultiDB) Find(ctx context.Context, col string, filter interface{},
	opts ...*options.FindOptions) (*mongo.Cursor, error) {
	var (
		cursor *mongo.Cursor
		err    error
	)

	for _, db := range m.dbs {
		cursor, err = db.Find(ctx, col, filter, opts...)
		if err != nil {
			log.Warnf("find from table %v failed: %v", col, err)
			continue
		} else {
			return cursor, nil
		}
	}

	return nil, err
}

func (m *MultiDB) Update(ctx context.Context, col string, filter, docs interface{}) (int, error) {
	var (
		upserted int
		err      error
	)

	for _, db := range m.dbs {
		upserted, err = db.Update(ctx, col, filter, docs)
		if err != nil {
			return upserted, err
		}
	}

	return upserted, nil
}

func (m *MultiDB) Delete(ctx context.Context, col string, filter interface{}) (int, error) {
	var (
		deleted int
		err     error
	)

	for _, db := range m.dbs {
		deleted, err = db.Delete(ctx, col, filter)
		if err != nil {
			return deleted, err
		}
	}

	return deleted, nil
}

func (m *MultiDB) Aggregate(ctx context.Context, col string, pipeline interface{}, res interface{}) error {
	var err error
	for _, db := range m.dbs {
		err = db.Aggregate(ctx, col, pipeline, res)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MultiDB) SetDbs(db *MgoDocDB) error {
	if db == nil {
		return xerrors.New("multidb setdbs: db is nil")
	}

	m.dbs = append(m.dbs, db)
	return nil
}
