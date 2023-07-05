package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/lib/limiter"
	lmodel "github.com/ipfs-force-community/londobell/racailum/segment/model"
)

var completeActorEventCmd = &cli.Command{
	Name: "complete-actorevent",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "end",
			Required: true,
			Usage:    "not included",
		},
		&cli.StringFlag{
			Name:     "url",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		actorEventCol := db.Collection("ActorEvent")

		js, err := ioutil.ReadFile("./cmd/bell/complete_actorevent.js")
		if err != nil {
			return err
		}

		startEpoch, endEpoch := cctx.Int64("start"), cctx.Int64("end")
		tsDone := startEpoch

		type Range struct {
			Start int64
			End   int64
		}

		part := make([]Range, 0)
		for tsDone < endEpoch {
			start := tsDone
			end := start + 2880*5
			if end > endEpoch {
				end = endEpoch
			}

			part = append(part, Range{Start: start, End: end})

			tsDone = end
		}

		sort.Slice(part, func(i, j int) bool {
			return part[i].Start > part[j].Start
		})

		lim := limiter.New(16)
		var ewg multierror.Group

		log.Infof("begin complet actorevent")
		starttime := time.Now()

		//for {
		for i := range part {
			i := i
			r := part[i]
			ewg.Go(func() error {
				if !lim.Acquire(context.TODO()) {
					return nil
				}

				defer func() {
					lim.Release(context.TODO())
				}()

				pipe, err := util.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, string(js))
				if err != nil {
					return err
				}

				cur, err := traceCol.Aggregate(context.TODO(), pipe)
				if err != nil {
					return err
				}

				var result []ActorEventRes
				var res []bson.M
				err = cur.All(context.TODO(), &res)
				if err != nil {
					return err
				}

				rawByte, err := json.Marshal(res)
				if err != nil {
					return err
				}

				err = json.Unmarshal(rawByte, &result)
				if err != nil {
					return err
				}

				var actorEvents = make([]*lmodel.ActorEvent, 0)
				for _, r := range result {
					if r.Cid == "bafy2bzaced5geaa6xd35ffiah3favxxuzbaqk6g2xnggm4dnntgcdwzkzrfy6" {
						fmt.Println("find")
					}

					var events []types.Event
					event, ok := r.Events.(primitive.Binary)
					if ok {
						err = json.Unmarshal(event.Data, &events)
						if err != nil {
							log.Error(err)
							return err
						}
					} else {
						event, ok := r.Events.(map[string]interface{})
						if !ok {
							log.Errorf("!ok")
							return fmt.Errorf("!ok")
						}

						binaryEvent, ok := event["$binary"].(map[string]interface{})
						if ok {
							binaryEventStr, ok := binaryEvent["base64"].(string)
							if ok {
								binaryEventByte, err := base64.StdEncoding.DecodeString(binaryEventStr)
								if err != nil {
									return err
								}
								err = json.Unmarshal(binaryEventByte, &events)
								if err != nil {
									return err
								}
							} else {
								log.Errorf("!ok")
								return fmt.Errorf("!ok")
							}
						} else {
							dataEventStr, ok := event["Data"].(string)
							if ok {
								dataEventByte, err := base64.StdEncoding.DecodeString(dataEventStr)
								if err != nil {
									return err
								}
								err = json.Unmarshal(dataEventByte, &events)
								if err != nil {
									return err
								}
							} else {
								log.Errorf("!ok")
								return fmt.Errorf("!ok")
							}
						}
					}

					for i, evt := range events {
						actorID, err := address.NewIDAddress(uint64(evt.Emitter))
						if err != nil {
							return fmt.Errorf("failed to create ID address: %w", err)
						}

						mcid, err := cid.Decode(r.Cid)
						if err != nil {
							return err
						}

						var signedCid cid.Cid
						signedCid, _ = cid.Decode(r.SignedCid)

						data, topics, ok := ethLogFromEvent(abi.ChainEpoch(r.Epoch), evt.Entries)
						if !ok {
							// not an eth event.
							log.Warnw("ethLogFromEvent not an eth event", "actorID", actorID, "mcid", mcid, "signedCid", signedCid)
							//continue //todo
						}

						logIndex := uint64(i)
						removed := false
						id := fmt.Sprintf("%v-%v", r.ID, logIndex)

						aet, err := lmodel.NewActorEvent2(actorID, abi.ChainEpoch(r.Epoch), mcid, signedCid, topics, data, logIndex, removed, id)
						if err != nil {
							log.Errorw("convert to model.ActorEvent", "actorID", actorID, "mcid", mcid, "signedCid", signedCid, "err", err.Error())
						} else {
							actorEvents = append(actorEvents, aet)
						}
					}
				}

				// insert into EvmInitCode
				var docs []interface{}
				for _, e := range actorEvents {
					docs = append(docs, e)
				}

				total := len(docs)
				if total > 0 {
					ires, err := actorEventCol.InsertMany(context.TODO(), docs, options.InsertMany().SetOrdered(false))
					if err != nil {
						if actualErr := extractActualMgoErrors(err); actualErr != nil {
							return actualErr
						}
					}

					log.Infof("part [%v, %v] inserted: %v/%v, elapsed: %v\n", r.Start, r.End, len(ires.InsertedIDs), total, time.Now().Sub(starttime).String())
					return nil
				}

				log.Infof("part [%v, %v] total 0, elapsed: %v\n", r.Start, r.End, time.Now().Sub(starttime).String())
				return nil
			})

		}

		if err := ewg.Wait(); err != nil {
			log.Errorf("falied: %v", err)
			//continue
			return err
		}

		log.Infof("all finished, elapsed: %v\n", time.Now().Sub(starttime).String())
		//break
		//}

		return nil
	},
}

type ActorEventRes struct {
	ID        string `bson:"_id" json:"_id"`
	Cid       string
	SignedCid string
	Epoch     int64
	Events    interface{}
}

func ethLogFromEvent(epoch abi.ChainEpoch, entries []types.EventEntry) (data []byte, topics []ethtypes.EthHash, ok bool) {
	elog := log.With("epoch", epoch)

	var (
		topicsFound      [4]bool
		topicsFoundCount int
		dataFound        bool
	)
	for _, entry := range entries {
		// Drop events with non-raw topics to avoid mistakes.
		if entry.Codec != cid.Raw {
			elog.Warnw("did not expect an event entry with a non-raw codec", "codec", entry.Codec, "key", entry.Key)
			return nil, nil, false
		}
		// Check if the key is t1..t4
		if len(entry.Key) == 2 && "t1" <= entry.Key && entry.Key <= "t4" {
			// '1' - '1' == 0, etc.
			idx := int(entry.Key[1] - '1')

			// Drop events with mis-sized topics.
			if len(entry.Value) != 32 {
				elog.Warnw("got an EVM event topic with an invalid size", "key", entry.Key, "size", len(entry.Value))
				return nil, nil, false
			}

			// Drop events with duplicate topics.
			if topicsFound[idx] {
				elog.Warnw("got a duplicate EVM event topic", "key", entry.Key)
				return nil, nil, false
			}
			topicsFound[idx] = true
			topicsFoundCount++

			// Extend the topics array
			for len(topics) <= idx {
				topics = append(topics, ethtypes.EthHash{})
			}
			copy(topics[idx][:], entry.Value)
		} else if entry.Key == "d" {
			// Drop events with duplicate data fields.
			if dataFound {
				elog.Warnw("got duplicate EVM event data")
				return nil, nil, false
			}

			dataFound = true
			data = entry.Value
		} else {
			// Skip entries we don't understand (makes it easier to extend things).
			// But we warn for now because we don't expect them.
			elog.Warnw("unexpected event entry", "key", entry.Key)
		}

	}

	// Drop events with skipped topics.
	if len(topics) != topicsFoundCount {
		elog.Warnw("EVM event topic length mismatch", "expected", len(topics), "actual", topicsFoundCount)
		return nil, nil, false
	}
	return data, topics, true
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
