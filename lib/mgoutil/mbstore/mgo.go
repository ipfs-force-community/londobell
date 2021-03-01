package mbstore

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var log = logging.Logger("fbstore")

// common errors
var (
	ErrNotImpl  = errors.New("not impl")
	ErrClosed   = errors.New("bstore closed")
	ErrReadOnly = errors.New("read only")
)

// MgoBlock is the mongo document format to represent a block
type MgoBlock struct {
	ID   string `bson:"_id"`
	B    []byte `bson:"b"`
	Size int    `bson:"s"`
	Time int64  `bson:"t"`
}

var _ blockstore.Blockstore = (*MgoBstore)(nil)
var bulkInsertOpt = options.InsertMany().SetOrdered(false)

// MgoReadOnly sets the read only option
func MgoReadOnly(ro bool) MgoStoreOption {
	return func(cfg *MgoStoreConfig) {
		cfg.readOnly = ro
	}
}

// MgoPutSync sets the put mode
func MgoPutSync(sync bool) MgoStoreOption {
	return func(cfg *MgoStoreConfig) {
		cfg.sync = sync
	}
}

// MgoStoreOption is the option applier
type MgoStoreOption func(*MgoStoreConfig)

// NewMgoBstore returns a *MgoBstore
func NewMgoBstore(ctx context.Context, dsn string, opts ...MgoStoreOption) (*MgoBstore, error) {
	cfg := MgoStoreConfig{
		connectTimeout:     5 * time.Second,
		gracefulTimeout:    10 * time.Second,
		statInterval:       1 * time.Minute,
		persistInterval:    5 * time.Second,
		persistIntervalMin: 2 * time.Second,
		persistThreshold:   512,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, err
	}

	connectCtx, connectCancel := context.WithTimeout(ctx, cfg.connectTimeout)
	defer connectCancel()

	err = client.Connect(connectCtx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	return &MgoBstore{
		client:    client,
		col:       client.Database("blockstore").Collection("blocks"),
		ctx:       ctx,
		ctxCancel: cancel,
		cfg:       cfg,
		pendingCh: make(chan []cid.Cid, cfg.persistThreshold/2),
		pending:   map[cid.Cid]blocks.Block{},
	}, nil
}

// MgoStoreConfig configures the mgo store
type MgoStoreConfig struct {
	connectTimeout     time.Duration
	gracefulTimeout    time.Duration
	statInterval       time.Duration
	persistInterval    time.Duration
	persistIntervalMin time.Duration
	persistThreshold   int
	sync               bool
	readOnly           bool
}

// MgoBstore is an implementation of blockstore.Blockstore based on mongodb
type MgoBstore struct {
	client    *mongo.Client
	col       *mongo.Collection
	ctx       context.Context
	ctxCancel context.CancelFunc
	cfg       MgoStoreConfig
	loopCtx   context.Context
	pendingCh chan []cid.Cid
	pending   map[cid.Cid]blocks.Block
	pendingMu sync.RWMutex
}

func (mb *MgoBstore) query(c cid.Cid) bson.D {
	return bson.D{{Key: "_id", Value: c.String()}}
}

func (mb *MgoBstore) block(c cid.Cid, b blocks.Block) MgoBlock {
	raw := b.RawData()
	return MgoBlock{
		ID:   c.String(),
		B:    raw,
		Size: len(raw),
		Time: time.Now().Unix(),
	}
}

func (mb *MgoBstore) tryPending(c cid.Cid) blocks.Block {
	mb.pendingMu.RLock()
	b, ok := mb.pending[c]
	mb.pendingMu.RUnlock()
	if ok {
		return b
	}

	return nil
}

// Run starts the inside loops
func (mb *MgoBstore) Run() {

	loopCtx, loopCancel := context.WithCancel(mb.ctx)
	mb.loopCtx = loopCtx

	go func() {

		ticker := time.NewTicker(mb.cfg.persistInterval)

		defer func() {
			ticker.Stop()
			loopCancel()
		}()

		pending := make([]cid.Cid, 0, mb.cfg.persistThreshold)
		lastPersist := time.Now()

	RUN_LOOP:
		for {
			must := false
			var sinceLast time.Duration

			select {
			case <-mb.ctx.Done():
				break RUN_LOOP

			case incoming := <-mb.pendingCh:
				pending = append(pending, incoming...)

			case t := <-ticker.C:
				sinceLast = t.Sub(lastPersist)
				must = sinceLast > mb.cfg.persistIntervalMin
			}

			pendingSize := len(pending)
			if (must && pendingSize > 0) || pendingSize >= mb.cfg.persistThreshold {
				log.Debugw("try persist blocks", "count", pendingSize, "must", must)
				toPersist := pending
				pending = make([]cid.Cid, 0, mb.cfg.persistThreshold)

				go mb.persist(mb.ctx, toPersist, false)

				lastPersist = time.Now()
			} else {
				log.Debugw("skip current persist round", "count", len(pending), "must", must, "tick-since", sinceLast)
			}
		}

		log.Warn("have to finish pending blocks before quitting the loop")
		pctx, pcancel := context.WithTimeout(context.Background(), mb.cfg.gracefulTimeout)
		defer pcancel()
		mb.persist(pctx, nil, true)
		log.Warn("finished pending blocks before quitting the loop")

	}()
	return
}

// Stop does some flushes and cleanups
func (mb *MgoBstore) Stop() {
	mb.ctxCancel()

	if mb.loopCtx != nil {
		select {
		case <-mb.loopCtx.Done():

		case <-time.After(mb.cfg.gracefulTimeout):
			log.Errorf("waited %s but loop still did not quit", mb.cfg.gracefulTimeout)
		}
	}
}

func (mb *MgoBstore) insertMany(ctx context.Context, blks []blocks.Block) error {
	if len(blks) == 0 {
		return nil
	}

	docs := make([]interface{}, 0, len(blks))
	for _, blk := range blks {
		docs = append(docs, mb.block(blk.Cid(), blk))
	}

	var inserted, total int

	start := time.Now()

	var merr error
	for total < len(docs) {
		last := total + mb.cfg.persistThreshold
		if last > len(docs) {
			last = len(docs)
		}

		part := docs[total:last]
		res, err := mb.col.InsertMany(ctx, part, bulkInsertOpt)
		if err != nil {
			if mbe, ok := err.(mongo.BulkWriteException); ok {
				for _, we := range mbe.WriteErrors {
					// duplicate _id
					if we.Code == 11000 {
						continue
					}

					merr = multierror.Append(merr, we)
				}
			} else {
				merr = multierror.Append(merr, err)
			}

		}

		if res != nil {
			inserted += len(res.InsertedIDs)
		}

		total = last
	}

	log.Debugw("sync blocks inserted", "insert", inserted, "total", total, "elapsed", time.Since(start).String())

	return merr
}

func (mb *MgoBstore) persist(ctx context.Context, cs []cid.Cid, flush bool) {
	if !flush && len(cs) == 0 {
		return
	}

	var docs []interface{}

	mb.pendingMu.RLock()
	if flush {
		docs = make([]interface{}, 0, len(mb.pending))
		cs = make([]cid.Cid, 0, len(mb.pending))
		for c := range mb.pending {
			docs = append(docs, mb.block(c, mb.pending[c]))
			cs = append(cs, c)
		}
		log.Debugw("flushing all pending blocks", "count", len(docs))

	} else {
		docs = make([]interface{}, 0, len(cs))
		for _, c := range cs {
			if b, ok := mb.pending[c]; ok {
				docs = append(docs, mb.block(c, b))
			}
		}

		log.Debugw("try to persist blocks", "cids", len(cs), "actual", len(docs))
	}
	mb.pendingMu.RUnlock()

	var inserted, total int

	start := time.Now()

	for total < len(docs) {
		last := total + mb.cfg.persistThreshold
		if last > len(docs) {
			last = len(docs)
		}

		part := docs[total:last]
		res, err := mb.col.InsertMany(ctx, part, bulkInsertOpt)
		if err != nil {
			if mbe, ok := err.(mongo.BulkWriteException); ok {
				for _, we := range mbe.WriteErrors {
					if we.Code == 11000 {
						continue
					}

					log.Errorf("failed insert in bulk write: %s", we.Error())
				}
			} else {
				log.Errorf("failed to insert %d blocks: %+v", len(part), err)
			}

		}

		if res != nil {
			inserted += len(res.InsertedIDs)
		}

		total = last
	}

	delCount := 0
	mb.pendingMu.Lock()
	for _, c := range cs {
		if _, ok := mb.pending[c]; ok {
			delete(mb.pending, c)
			delCount++
		}
	}
	mb.pendingMu.Unlock()

	log.Debugw("blocks inserted", "insert", inserted, "total", total, "evict", delCount, "elapsed", time.Since(start).String())
}

// DeleteBlock impls blockstore.Blockstore
func (mb *MgoBstore) DeleteBlock(c cid.Cid) error {
	if mb.cfg.readOnly {
		return nil
	}

	// We simply DO NOT DELETE blocks from store
	_, err := mb.col.DeleteOne(mb.ctx, mb.query(c))
	return err
}

// Has impls blockstore.Blockstore
func (mb *MgoBstore) Has(c cid.Cid) (bool, error) {
	if b := mb.tryPending(c); b != nil {
		return true, nil
	}

	count, err := mb.col.CountDocuments(mb.ctx, mb.query(c))
	return count != 0, err
}

// Get impls blockstore.Blockstore
func (mb *MgoBstore) Get(c cid.Cid) (blocks.Block, error) {
	if b := mb.tryPending(c); b != nil {
		return b, nil
	}

	var b MgoBlock
	err := mb.col.FindOne(mb.ctx, mb.query(c)).Decode(&b)
	if err == nil {
		return blocks.NewBlockWithCid(b.B, c)
	}

	if err == mongo.ErrNoDocuments {
		return nil, blockstore.ErrNotFound
	}

	return nil, err
}

// GetSize impls blockstore.Blockstore
func (mb *MgoBstore) GetSize(c cid.Cid) (int, error) {
	return 0, ErrNotImpl
}

// Put impls blockstore.Blockstore
func (mb *MgoBstore) Put(b blocks.Block) error {
	if mb.cfg.readOnly {
		return nil
	}

	return mb.PutMany([]blocks.Block{b})
}

// PutMany impls blockstore.Blockstore
func (mb *MgoBstore) PutMany(bs []blocks.Block) error {
	if mb.cfg.readOnly {
		return nil
	}

	select {
	case <-mb.ctx.Done():
		return ErrClosed

	default:

	}

	if mb.cfg.sync {
		return mb.insertMany(mb.ctx, bs)
	}

	notExists := make([]cid.Cid, 0, len(bs))
	mb.pendingMu.Lock()
	for i := range bs {
		b := bs[i]
		c := b.Cid()
		if _, ok := mb.pending[c]; !ok {
			mb.pending[c] = b
			notExists = append(notExists, c)
		}
	}
	mb.pendingMu.Unlock()

	if len(notExists) == 0 {
		return nil
	}

	log.Debugw("put many blocks", "count", len(notExists))

	go func() {
		select {
		case <-mb.ctx.Done():

		case mb.pendingCh <- notExists:

		default:

		}
	}()

	return nil
}

// AllKeysChan impls blockstore.Blockstore
func (mb *MgoBstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, ErrNotImpl
}

// HashOnRead impls blockstore.Blockstore
func (mb *MgoBstore) HashOnRead(enabled bool) {

}
