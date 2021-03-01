package headsub

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/dtynn/londobell/lib/mgoutil/mmetads"
)

var log = logging.Logger("headsub")

type changeEvent struct {
	FullDocument mmetads.MgoMetaItem `bson:"fullDocument"`
}

// New returns a head notifier based on mongo metads
func New(cli *mongo.Client) (*HeadSub, error) {
	return &HeadSub{
		col: cli.Database(mmetads.DBName).Collection(mmetads.ColName),
	}, nil
}

// HeadSub subscribe head changes from metads on mongo
type HeadSub struct {
	col *mongo.Collection
}

// Sub returns a chan of types.TipSetKey which informs the receiver about the head change of the chain
func (m *HeadSub) Sub(ctx context.Context) (<-chan types.TipSetKey, error) {

	opt := options.ChangeStream().SetFullDocument(options.UpdateLookup)
	pipe := mongo.Pipeline{
		{
			{
				Key: "$match",
				Value: bson.D{
					{
						Key:   "documentKey._id",
						Value: "/head",
					},
					{
						Key:   "operationType",
						Value: "replace",
					},
				}},
		},
	}

	watch, err := m.col.Watch(ctx, pipe, opt)
	if err != nil {
		return nil, err
	}

	tskCh := make(chan types.TipSetKey, 1)

	go func() {
		defer func() {
			close(tskCh)
			log.Info("head change stream loop stop")
		}()

		log.Info("head change stream loop start")

		for {
			select {
			case <-ctx.Done():
				return

			default:

			}

			if err := m.watch(ctx, watch, tskCh); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Errorf("error occurs in head change stream: %s", err)
				}
			}

			opt = opt.SetResumeAfter(watch.ResumeToken())

		REWATCH:
			for {
				select {
				case <-time.After(5 * time.Second):
					watch, err = m.col.Watch(ctx, pipe, opt)
					if err == nil {
						break REWATCH
					}

					log.Errorf("unable to establish head change stream: %s", err)

				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return tskCh, nil
}

func (m *HeadSub) watch(ctx context.Context, watch *mongo.ChangeStream, ch chan types.TipSetKey) error {
	defer watch.Close(ctx)

	var cancel context.CancelFunc
	for watch.Next(ctx) {
		var event changeEvent
		err := watch.Decode(&event)
		if err != nil {
			log.Errorf("unable to unmarshal head change event: %s", err)
			continue
		}

		var cids []cid.Cid
		if err := json.Unmarshal(event.FullDocument.B, &cids); err != nil {
			log.Errorf("unable to unmarshal into cids: %s", err)
			continue
		}

		if len(cids) == 0 {
			log.Warn("get empty tipset key")
			continue
		}

		if cancel != nil {
			cancel()
		}

		tsk := types.NewTipSetKey(cids...)
		sendctx, sendcancel := context.WithCancel(ctx)
		cancel = sendcancel
		go delaySend(sendctx, ch, tsk)

	}

	return watch.Err()
}

func delaySend(ctx context.Context, ch chan types.TipSetKey, tsk types.TipSetKey) {
	slog := log.With("tsk", tsk)

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")
		return

	case <-timer.C:

	}

	wait := time.NewTimer(time.Second)
	defer wait.Stop()

	select {
	case <-ctx.Done():
		slog.Debug("aborted")

	case ch <- tsk:
		slog.Debug("sent")

	case <-wait.C:
		slog.Debug("out chan is full")
	}
}
