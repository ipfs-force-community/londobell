package mgoutil

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"
)

// NewMgoDocDB returns an instance of MgoDocDB
func NewMgoDocDB(ctx context.Context, cli *mongo.Client, db *mongo.Database) (*MgoDocDB, error) {
	mdb := &MgoDocDB{
		cli:       cli,
		db:        db,
		insertOpt: options.InsertMany().SetOrdered(false),
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
