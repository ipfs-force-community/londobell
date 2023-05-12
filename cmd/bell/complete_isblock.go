package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/hashicorp/go-multierror"

	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

var completeIsBlockCmd = &cli.Command{
	Name: "complete-isblock",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start", //2849393
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "end", //2883335
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
		&cli.StringFlag{
			Name:     "type",
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
		var js []byte
		switch cctx.String("type") {
		case "1":
			js, err = ioutil.ReadFile("./cmd/bell/merge_isblock1.js")
			if err != nil {
				return err
			}
		case "2":
			js, err = ioutil.ReadFile("./cmd/bell/merge_isblock2.js")
			if err != nil {
				return err
			}
		case "3":
			js, err = ioutil.ReadFile("./cmd/bell/merge_isblock3.js")
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid type: %v", cctx.String("type"))
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

		log.Infof("begin complet value")
		starttime := time.Now()

		for {
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

					pipe, err := aggregators.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, string(js))
					if err != nil {
						return err
					}

					cur, err := traceCol.Aggregate(context.TODO(), pipe)
					if err != nil {
						return err
					}

					var res []bson.M
					err = cur.All(context.TODO(), &res)
					if err != nil {
						return err
					}

					log.Infof("part [%v, %v] elapsed: %v\n", r.Start, r.End, time.Now().Sub(starttime).String())
					return nil
				})

			}

			if err := ewg.Wait(); err != nil {
				log.Errorf("falied: %v", err)
				continue
			}

			log.Infof("all finished, elapsed: %v\n", time.Now().Sub(starttime).String())
			break
		}

		return nil
	},
}
