package mdict

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dtynn/londobell/common"
)

var _ common.ChainDict = (*Dict)(nil)

type dictDoc struct {
	Namespace string `bson:"_id"`
	Entries   []string
}

type cachedItem struct {
	doc dictDoc
	e2i map[string]int
}

// NewDict returns a *Dict instance
func NewDict(db *mongo.Database) (*Dict, error) {
	return &Dict{
		col:   db.Collection("dict"),
		cache: map[string]*cachedItem{},
	}, nil
}

// Dict is the implementation of common.ChainDict based on mongodb
type Dict struct {
	col *mongo.Collection

	cacheMutex sync.RWMutex
	cache      map[string]*cachedItem
}

// AddEnum add entries into specific namespace
func (d *Dict) AddEnum(ctx context.Context, ns string, entry ...string) error {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	item, ok := d.cache[ns]
	if !ok {
		var doc dictDoc
		if err := d.col.FindOneAndUpdate(ctx, bson.M{"_id": ns}, bson.M{"$addToSet": bson.M{"Entries": bson.M{"$each": entry}}}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&doc); err != nil {
			return fmt.Errorf("get original item: %w", err)
		}

		d.cache[ns] = makeCachedItem(doc)

		return nil
	}

	deduped := make([]string, 0, len(entry))
	before := len(item.e2i)

	for i := range entry {
		name := entry[i]
		if _, found := item.e2i[name]; found {
			continue
		}

		deduped = append(deduped, name)
	}

	if len(deduped) == 0 {
		return nil
	}

	if err := d.col.FindOneAndUpdate(ctx, bson.M{"_id": ns}, bson.M{"$addToSet": bson.M{"Entries": bson.M{"$each": deduped}}}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&item.doc); err != nil {
		return fmt.Errorf("add entries: %w", err)
	}

	if after := len(item.doc.Entries); after <= before {
		return fmt.Errorf("entry lenth mismatched: before %d, after %d", before, after)
	}

	for i := before; i < len(item.doc.Entries); i++ {
		name := item.doc.Entries[i]
		item.e2i[name] = i
	}

	return nil
}

// LookupEnum tries to find the index of the entry in the given namespace
func (d *Dict) LookupEnum(ctx context.Context, ns string, entry string) (int, error) {
	var e2i map[string]int

	d.cacheMutex.RLock()
	cached, ok := d.cache[ns]
	if ok {
		e2i = cached.e2i
	}
	d.cacheMutex.RUnlock()

	if e2i == nil {
		var doc dictDoc
		err := d.col.FindOne(ctx, bson.M{"_id": ns}).Decode(&doc)
		if err != nil {
			return 0, fmt.Errorf("find dict doc for %s: %w", ns, err)
		}

		d.cacheMutex.Lock()
		d.cache[ns] = makeCachedItem(doc)
		e2i = d.cache[ns].e2i
		d.cacheMutex.Unlock()
	}

	idx, found := e2i[entry]
	if !found {
		return 0, fmt.Errorf("not found")
	}

	return idx, nil

}

func makeCachedItem(doc dictDoc) *cachedItem {
	cached := &cachedItem{
		doc: doc,
		e2i: map[string]int{},
	}

	for i := range cached.doc.Entries {
		name := cached.doc.Entries[i]
		cached.e2i[name] = i
	}

	return cached
}
