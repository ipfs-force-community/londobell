package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/lib/limiter"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hashicorp/go-multierror"

	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

var completeInitCodeCmd = &cli.Command{
	Name: "complete-initcode",
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
		&cli.StringFlag{
			Name:     "type",
			Required: true,
			Usage:    "new or old",
		},
		&cli.StringFlag{
			Name:  "api-url",
			Usage: "ws://127.0.0.1:1234/rpc/v0",
		},
	},
	Action: func(cctx *cli.Context) error {
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		evmInitCodeCol := db.Collection("EvmInitCode")
		var js []byte
		if cctx.String("type") == "new" {
			js, err = ioutil.ReadFile("./cmd/bell/add_initcode_new.js")
			if err != nil {
				return err
			}
		} else if cctx.String("type") == "old" {
			js, err = ioutil.ReadFile("./cmd/bell/add_initcode_old.js")
			if err != nil {
				return err
			}
		}

		api, closer, err := adapter.GetFullNodeAPI(context.TODO(), cctx.String("api-url"))
		defer closer()
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

		log.Infof("begin complete initcode")
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

					var res []InitCodeRes
					err = cur.All(context.TODO(), &res)
					if err != nil {
						return err
					}

					var initCodes = make([]InitCode, len(res))
					for i, r := range res {
						inticode, err := getInitCodeFromParams(r.InitCode)
						if err != nil {
							return err
						}

						//actor, err := address.NewIDAddress(r.ActorID)
						//if err != nil {
						//	return err
						//}

						actor, err := address.NewFromString(buildnet.NetPrefix + r.ActorID)
						if err != nil {
							return err
						}

						actorID, err := api.StateLookupID(context.TODO(), actor, types.EmptyTSK)
						if err != nil {
							return err
						}

						initCodes[i] = InitCode{Epoch: abi.ChainEpoch(r.Epoch), ActorID: actorID.String()[1:], InitCode: inticode}
					}

					// insert into EvmInitCode
					var docs []interface{}
					for _, initcode := range initCodes {
						d := bson.D{
							{Key: "_id", Value: initcode.ActorID},
							{Key: "InitCode", Value: initcode.InitCode},
							{Key: "Epoch", Value: initcode.Epoch},
						}

						docs = append(docs, d)
					}

					total := len(docs)
					if total > 0 {
						ires, err := evmInitCodeCol.InsertMany(context.TODO(), docs, options.InsertMany().SetOrdered(false))
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
				continue
			}

			log.Infof("all finished, elapsed: %v\n", time.Now().Sub(starttime).String())
			break
		}

		return nil
	},
}

type InitCodeRes struct {
	ID       string `bson:"_id"`
	Epoch    int64
	ActorID  string
	InitCode interface{}
}

type InitCode struct {
	Epoch    abi.ChainEpoch
	ActorID  string
	InitCode string
}

func getInitCodeFromParams(initCodeParams interface{}) (string, error) {
	// 兼容$binary
	var (
		ParamsByte []byte
		err        error
	)
	params, ok := initCodeParams.(map[string]interface{})
	if !ok {
		params, ok := initCodeParams.(primitive.Binary)
		if !ok {
			params, ok := initCodeParams.(primitive.D)
			if !ok {
				return "", fmt.Errorf("invalid type of params")
			}

			paramsM := params.Map()
			if len(paramsM) == 0 {
				return "", fmt.Errorf("invalid type primitive.D")
			}

			paramsE, ok := paramsM["$binary"].(primitive.D)
			if !ok {
				return "", fmt.Errorf("invalid type primitive.E")
			}

			paramsEM := paramsE.Map()
			if len(paramsEM) == 0 {
				return "", fmt.Errorf("invalid type paramsEM")
			}

			paramsEStr, ok := paramsEM["base64"].(string)
			if ok {
				ParamsByte, err = base64.StdEncoding.DecodeString(paramsEStr)
				if err != nil {
					return "", err
				}
			}
		} else {
			ParamsByte = params.Data
		}
	}

	binaryParams, ok := params["$binary"].(map[string]interface{})
	if ok {
		binaryParamsStr, ok := binaryParams["base64"].(string)
		if ok {
			ParamsByte, err = base64.StdEncoding.DecodeString(binaryParamsStr)
			if err != nil {
				return "", err
			}
		}
	} else {
		dataParamsStr, ok := params["Data"].(string)
		if ok {
			ParamsByte, err = base64.StdEncoding.DecodeString(dataParamsStr)
			if err != nil {
				return "", err
			}
		}
	}

	//var initCode abi.CborBytes
	//err = initCode.UnmarshalCBOR(bytes.NewReader(ParamsByte))
	//if err != nil {
	//	return "", err
	//}

	return hex.EncodeToString(ParamsByte), nil
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
