package main

import (
	"context"
	"io/ioutil"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/lib/limiter"
)

var completeProposalCmd = &cli.Command{
	Name: "complete-dealproposal",
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
			Name:     "nodeconfig",
			Usage:    "The location of the node configuration, eg: ./config.json(api: token)",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		if err := util.ParseNodes(cctx.String("nodeconfig")); err != nil {
			return err
		}

		fullnode.API = fullnode.NewAppropriateAPI(util.Nodes)
		err := fullnode.API.Choose(ctx)
		if err != nil {
			return err
		}

		api := fullnode.API.GetAppropriateAPI()

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		dealProposal := db.Collection("DealProposal")

		js, err := ioutil.ReadFile("./cmd/bell/get_deal.js")
		if err != nil {
			return err
		}

		js2, err := ioutil.ReadFile("./cmd/bell/complete_dealproposal.js")
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

		log.Infof("begin complet dealproposal")
		starttime := time.Now()

		actorIDMap := make(map[address.Address]address.Address)
		var alk sync.RWMutex

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

				cur, err := dealProposal.Aggregate(context.TODO(), pipe)
				if err != nil {
					return err
				}

				var res []DealProposalRes
				err = cur.All(context.TODO(), &res)
				if err != nil {
					return err
				}

				for _, r := range res {
					var (
						providerID address.Address
						clientID   address.Address
					)

					alk.RLock()
					provider, err := address.NewFromString(r.Provider)
					if err != nil {
						return err
					}

					client, err := address.NewFromString(r.Client)
					if err != nil {
						return err
					}

					pID, ok := actorIDMap[provider]
					alk.RUnlock()
					if ok {
						providerID = pID
					} else {
						providerID, err = api.StateLookupID(ctx, provider, types.EmptyTSK)
						if err != nil {
							return err
						}

						alk.Lock()
						actorIDMap[provider] = providerID
						alk.Unlock()
					}

					cID, ok := actorIDMap[client]
					alk.RUnlock()
					if ok {
						clientID = cID
					} else {
						clientID, err = api.StateLookupID(ctx, client, types.EmptyTSK)
						if err != nil {
							return err
						}

						alk.Lock()
						actorIDMap[client] = clientID
						alk.Unlock()
					}

					pipe2, err := util.Parse(model.Ctx{ProviderID: providerID.String()[1:], ClientID: clientID.String()[1:]}, string(js2))
					if err != nil {
						return err
					}

					cur2, err := dealProposal.Aggregate(context.TODO(), pipe2)
					if err != nil {
						return err
					}

					var res []bson.M
					err = cur2.All(context.TODO(), &res)
					if err != nil {
						return err
					}

					log.Infof("dealID %v merge", r.ID)
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

type DealProposalRes struct {
	ID    string `bson:"_id" json:"_id"`
	Epoch int64
	//PieceCID             string
	//PieceSize            int64
	//VerifiedDeal         bool
	Client   string
	Provider string
	//Label                interface{}
	//StartEpoch           int64
	//EndEpoch             int64
	//StoragePricePerEpoch string
	//ProviderCollateral   string
	//ClientCollateral     string
}
