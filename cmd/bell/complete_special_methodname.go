package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/lib/limiter"
)

//todo:2878193-2892593 fix hk
var completeSpecialMethodNameCmd = &cli.Command{
	Name: "complete-special-methodname",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:     "start", //2849393
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "end", //2929445
			Required: true,
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
			Usage:    "complete or fix",
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
		if cctx.String("type") == "complete" {
			js, err = ioutil.ReadFile("./cmd/bell/merge_special_methodname.js")
			if err != nil {
				return err
			}
		} else if cctx.String("type") == "fix" {
			js, err = ioutil.ReadFile("./cmd/bell/fix_special_methodname.js")
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown type: %v", cctx.String("type"))
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

		log.Infof("begin complete special methoname")
		starttime := time.Now()

		log.Infof("parts: %v", part)

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

			//	2500422-2849164
			//[{2500422 2586822} {2586822 2673222} {2673222 2759622} {2759622 2846022} {2846022 2849164}]
			//{2500422 2514822} {2514822 2529222} {2529222 2543622} {2543622 2558022} {2558022 2572422} {2572422 2586822} {2586822 2601222} {2601222 2615622} {2615622 2630022} {2630022 2644422} {2644422 2658822} ...
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
