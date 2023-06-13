package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/ipfs-force-community/londobell/dep"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/build"

	"github.com/dtynn/dix"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/lib/limiter"

	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/londobell/common"
)

var completeEthHashCmd = &cli.Command{
	Name: "complete-ethhash",
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
		&cli.BoolFlag{
			Name:     "local",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()

		var components struct {
			fx.In
			CS common.ChainStore
			SM common.StateManager
		}

		stopper, err := dix.New(ctx,
			Bell2(cctx, fxlog, cctx.Bool("local"), &components),
			dep.InjectFullNode(cctx),
			//dep.InjectRepoPath(cctx),
		)

		defer stopper(ctx) // nolint: errcheck
		if err != nil {
			return err
		}

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cctx.String("url")))
		if err != nil {
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		ethhashCol := db.Collection("EthHash")

		js, err := ioutil.ReadFile("./cmd/bell/merge_ethhash.js")
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

		log.Infof("begin complete ethhash")
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

				pipe, err := aggregators.Parse(model.Ctx{StartEpoch: r.Start, EndEpoch: r.End}, string(js))
				if err != nil {
					return err
				}

				cur, err := traceCol.Aggregate(context.TODO(), pipe)
				if err != nil {
					return err
				}

				var res []EthHashRes
				err = cur.All(context.TODO(), &res)
				if err != nil {
					return err
				}

				var ethHashs = make([]EthHash, 0)
				for _, r := range res {
					scid, err := cid.Decode(r.Cid)
					if err != nil {
						return err
					}

					smsg, err := components.CS.GetSignedMessage(ctx, scid)
					if err != nil {
						return err
					}

					if smsg.Signature.Type != crypto.SigTypeDelegated {
						return fmt.Errorf("not crypto.SigTypeDelegated: %v", smsg.Signature.Type)
					}

					ts, err := components.CS.LoadTipSet(ctx, types.EmptyTSK)
					if err != nil {
						return err
					}

					ethhash, err := newEthTxFromSignedMessage(ctx, smsg, ts, components.SM)
					if err != nil {
						return err
					}

					ethHashs = append(ethHashs, EthHash{Hash: ethhash, Cid: scid, Epoch: abi.ChainEpoch(r.Epoch)})
					log.Infof("ethhash, hash: %v, cid: %v, epoch: %v", ethhash.String(), scid.String(), r.Epoch)
				}

				// insert into EvmInitCode
				var docs []interface{}

				for _, e := range ethHashs {
					docs = append(docs, e)
				}

				total := len(docs)
				if total > 0 {
					ires, err := ethhashCol.InsertMany(context.TODO(), docs, options.InsertMany().SetOrdered(false))
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

type EthHashRes struct {
	Cid   string
	Epoch int64
}

type EthHash struct {
	Hash  ethtypes.EthHash `bson:"_id"`
	Cid   cid.Cid
	Epoch abi.ChainEpoch
}

func newEthTxFromSignedMessage(ctx context.Context, smsg *types.SignedMessage, ts *types.TipSet, sm common.StateManager) (ethtypes.EthHash, error) {
	var tx ethtypes.EthTx
	var err error

	if smsg.Signature.Type == crypto.SigTypeDelegated {
		tx, err = ethtypes.EthTxFromSignedEthMessage(smsg)
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to convert from signed message: %w", err)
		}

		tx.Hash, err = tx.TxHash()
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to calculate hash for ethTx: %w", err)
		}

		fromAddr, err := lookupEthAddress(ctx, smsg.Message.From, ts, sm)
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
		}

		tx.From = fromAddr
	} else if smsg.Signature.Type == crypto.SigTypeSecp256k1 { // Secp Filecoin Message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), ts, sm)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	} else { // BLS Filecoin message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), ts, sm)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Message.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	}

	return tx.Hash, nil
}

func lookupEthAddress(ctx context.Context, addr address.Address, ts *types.TipSet, sm common.StateManager) (ethtypes.EthAddress, error) {
	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err == nil && !ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	if actor, err := sm.LoadActor(ctx, addr, ts); err != nil {
		return ethtypes.EthAddress{}, err
	} else if actor.Address != nil {
		if ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(*actor.Address); err == nil && !ethAddr.IsMaskedID() {
			return ethAddr, nil
		}
	}

	if err == nil && ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	idAddr, err := sm.LookupID(ctx, addr, ts)
	if err != nil {
		return ethtypes.EthAddress{}, err
	}
	return ethtypes.EthAddressFromFilecoinAddress(idAddr)
}

func ethTxFromNativeMessage(ctx context.Context, msg *types.Message, ts *types.TipSet, sm common.StateManager) ethtypes.EthTx {
	// We don't care if we error here, conversion is best effort for non-eth transactions
	from, _ := lookupEthAddress(ctx, msg.From, ts, sm)
	to, _ := lookupEthAddress(ctx, msg.To, ts, sm)
	return ethtypes.EthTx{
		To:                   &to,
		From:                 from,
		Nonce:                ethtypes.EthUint64(msg.Nonce),
		ChainID:              ethtypes.EthUint64(build.Eip155ChainId),
		Value:                ethtypes.EthBigInt(msg.Value),
		Type:                 ethtypes.Eip1559TxType,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         ethtypes.EthBigInt(msg.GasFeeCap),
		MaxPriorityFeePerGas: ethtypes.EthBigInt(msg.GasPremium),
		AccessList:           []ethtypes.EthHash{},
	}
}
