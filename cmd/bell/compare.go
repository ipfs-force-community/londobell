package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/lib/mgoutil"
	"github.com/ipfs-force-community/londobell/racailum/segment"
)

//只看块消息
//给定一段高度，查出该高度内的所有消息cid和块cid，和数据库比较

var compareCmd = &cli.Command{
	Name:  "compare",
	Usage: "compare data between db and chain",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "startkey",
			Required: true,
			Usage:    "tipsetkey for start epoch, Separated by ',' ",
		},
		&cli.StringFlag{
			Name:     "endkey",
			Required: true,
			Usage:    "tipsetkey for end epoch, Separated by ',' ",
		},
	},
	Action: func(cctx *cli.Context) error {
		di := struct {
			fx.In
			CS     common.ChainStore
			SegMgr *segment.Manager
		}{}

		stopper, err := dix.New(
			cctx.Context,
			dep.Bell(cctx.Context, fxlog, &di),
			dep.InjectFullNode(cctx),
			dep.InjectRepoPath(cctx),
		)
		if err != nil {
			return err
		}

		defer stopper(cctx.Context) // nolint: errcheck

		// 连接数据库
		info, ihas, err := di.SegMgr.LoadInfo(cctx.String("name"))
		if err != nil {
			return fmt.Errorf("load info: %w", err)
		}

		if !ihas {
			return fmt.Errorf("info not found")
		}

		rcli, err := mgoutil.Connect(cctx.Context, info.DSN.Read)
		if err != nil {
			return fmt.Errorf("connect to read db: %w", err)
		}

		// 比较tipset
		stsk, err := parsetTipSetKey(cctx.String("startkey"))
		if err != nil {
			return err
		}

		etsk, err := parsetTipSetKey(cctx.String("endkey"))
		if err != nil {
			return err
		}

		startTs, err := di.CS.LoadTipSet(cctx.Context, stsk)
		if err != nil {
			return err
		}

		endTs, err := di.CS.LoadTipSet(cctx.Context, etsk)
		if err != nil {
			return err
		}

		length := int64(endTs.Height() - startTs.Height() + 1)
		tss := make([]*types.TipSet, 0, length)
		tss = append(tss, endTs)

		var startEpoch, endEpoch = endTs.Height(), endTs.Height()

		parentTs := endTs
		for i := int64(1); i < length; i++ {
			if endTs.Height() == 0 {
				break
			}

			curEpoch := parentTs.Height() - 1
			curTs, err := di.CS.LoadTipSet(cctx.Context, parentTs.Parents())
			if err != nil {
				return err
			}

			// 跳过空块
			if curTs.Height() != curEpoch {
				i++
			}

			tss = append(tss, curTs)
			startEpoch = curTs.Height()
			parentTs = curTs
		}

		log.Infof("get [%d, %d] tipset: %+v", startEpoch, endEpoch, tss)

		database := rcli.Database(cctx.String("name"))
		tipsetResults, err := getTipset(cctx.Context, startEpoch, endEpoch, database)
		if err != nil {
			return fmt.Errorf("get tipset from db err: %w", err)
		}

		tipsetKeys := make([]string, 0, len(tipsetResults))
		for _, tipset := range tipsetResults {
			tipsetKeys = append(tipsetKeys, tipset.Cids...)
		}

		log.Infof("get [%d, %d] tipsetResults from db: %+v", startEpoch, endEpoch, tipsetKeys)

		tequal := compareTipset(tss, tipsetResults)
		if !tequal {
			log.Errorw("compare tipset not equal!", "startepoch:", startEpoch, "endepoch:", endEpoch)
		} else {
			log.Infow("compare tipset equal!", "startepoch:", startEpoch, "endepoch:", endEpoch)
		}

		// 比较message
		allBmsgs := make([]*types.Message, 0)
		allSmsgs := make([]*types.SignedMessage, 0)

		for _, ts := range tss {
			for _, b := range ts.Blocks() {
				bmsgs, smsgs, err := di.CS.MessagesForBlock(cctx.Context, b)
				if err != nil {
					return fmt.Errorf("get message for block [%v] err: %w ", b.Cid(), err)
				}

				allBmsgs = append(allBmsgs, bmsgs...)
				allSmsgs = append(allSmsgs, smsgs...)
			}
		}

		bmessages := struct {
			BlsMessages   []*types.Message
			SecpkMessages []*types.SignedMessage
		}{}

		bmessages.BlsMessages = allBmsgs
		bmessages.SecpkMessages = allSmsgs

		out, err := json.MarshalIndent(bmessages, "", "  ")
		if err != nil {
			return err
		}
		log.Infof("get [%d, %d] block messages: %s", startEpoch, endEpoch, string(out))

		messageResultsMap, err := getMessageMap(cctx.Context, startEpoch, endEpoch, database)
		if err != nil {
			return fmt.Errorf("get message from db err: %w", err)
		}

		messageResults := make([]Message, 0, len(messageResultsMap))
		for _, message := range messageResultsMap {
			messageResults = append(messageResults, message)
		}

		log.Infof("get [%d, %d] messageResultsMap from db: %+v", startEpoch, endEpoch, messageResults)

		mequal := compareMessage(allBmsgs, allSmsgs, messageResultsMap)
		if !mequal {
			log.Errorw("compare message not equal!", "startepoch:", startEpoch, "endepoch:", endEpoch)
		} else {
			log.Infow("compare message equal!", "startepoch:", startEpoch, "endepoch:", endEpoch)
		}

		return nil
	},
}

type Tipset struct {
	Height       int64    `bson:"_id"`
	Cids         []string `bson:"Cids"`
	MinTimestamp uint64   `bson:"MinTimestamp"`
	ChildEpoch   int64    `bson:"ChildEpoch"`
	State        string   `bson:"State"`
	Receipts     string   `bson:"Receipts"`
	Weight       string   `bson:"Weight"`
	BaseFee      string   `bson:"BaseFee"`
}

// 表tipset和链比较
func getTipset(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, database *mongo.Database) (tipsetResults []*Tipset, err error) {
	tipsetCollection := database.Collection("Tipset")
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{
					Key: "_id", Value: bson.D{{Key: "$gte", Value: startEpoch}, {Key: "$lte", Value: endEpoch}},
				},
			},
		},
	}
	sortStage := bson.D{
		{
			Key: "$sort", Value: bson.D{{Key: "_id", Value: -1}},
		},
	}
	cursor, err := tipsetCollection.Aggregate(ctx, mongo.Pipeline{matchStage, sortStage})
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &tipsetResults); err != nil {
		return nil, err
	}

	return tipsetResults, nil
}

// todo:开多个协程比较一段高度
func compareTipset(tss []*types.TipSet, tipsetResults []*Tipset) bool {
	if len(tss) != len(tipsetResults) {
		return false
	}
	for i, ts := range tss {
		if !equalForTipset(ts, tipsetResults[i]) {
			return false
		}
	}

	return true
}

// 1656366是空块
func equalForTipset(ts *types.TipSet, res *Tipset) bool {
	if ts == nil && res == nil {
		return true
	}

	if ts == nil || res == nil {
		return false
	}

	if int64(ts.Height()) != res.Height {
		return false
	}

	if len(ts.Cids()) != len(res.Cids) {
		return false
	}

	for i, cid := range ts.Cids() {
		if cid.String() != res.Cids[i] {
			return false
		}
	}

	return true
}

type Message struct {
	Cid        string `bson:"_id"`
	SignedCid  string `bson:"SignedCid"`
	Version    uint64 `bson:"Version"`
	To         string `bson:"To"`
	From       string `bson:"From"`
	Nonce      uint64 `bson:"Nonce"`
	Value      string `bson:"Value"`
	GasLimit   int64  `bson:"GasLimit"`
	GasFeeCap  string `bson:"GasFeeCap"`
	GasPremium string `bson:"GasPremium"`
	Method     uint64 `bson:"Method"`
	Params     []byte `bson:"Params"`
}

func getMessageMap(ctx context.Context, startEpoch, endEpoch abi.ChainEpoch, database *mongo.Database) (map[string]Message, error) {
	messageCollection := database.Collection("Message")
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{
					Key: "Detail.PackedHeight", Value: bson.D{{Key: "$gte", Value: startEpoch}, {Key: "$lte", Value: endEpoch}},
				},
			},
		},
	}
	cursor, err := messageCollection.Aggregate(ctx, mongo.Pipeline{matchStage})
	if err != nil {
		return nil, err
	}

	var messageResults []Message
	if err = cursor.All(ctx, &messageResults); err != nil {
		return nil, err
	}

	messageResultsMap := make(map[string]Message, len(messageResults))
	for _, message := range messageResults {
		if message.SignedCid != "" {
			log.Infof("message.SignedCid is not null, message.SignedCid: %v", message.SignedCid)
			messageResultsMap[message.SignedCid] = message
		} else {
			log.Infof("message.SignedCid is null, message.Cid: %v", message.Cid)
			messageResultsMap[message.Cid] = message
		}
	}
	return messageResultsMap, nil
}

func compareMessage(bmsgs []*types.Message, smsgs []*types.SignedMessage, messageResultsMap map[string]Message) bool {
	notFoundMessages := make([]types.Message, 0)
	notFoundSignedMessages := make([]types.SignedMessage, 0)
	for _, bmsg := range bmsgs {
		if message, ok := messageResultsMap[bmsg.Cid().String()]; ok {
			if !equalForMessage(*bmsg, message) {
				log.Errorf("compare bmsgs failed, not equal! bmsg: %+v, message: %+v", *bmsg, message)
				return false
			}
		} else {
			// 暂存未发现的消息 (可能是块消息中未被执行的消息)
			notFoundMessages = append(notFoundMessages, *bmsg)
		}
	}

	for _, notfoundmsg := range notFoundMessages {
		found := false
		for _, message := range messageResultsMap {
			if message.From == strings.TrimPrefix(notfoundmsg.From.String(), string(notfoundmsg.From.String()[0])) && message.Nonce == notfoundmsg.Nonce {
				found = true
				break
			}
		}

		if !found {
			log.Errorf("compare bmsgs failed, message not found! notfoundmsg: %+v, notfoundmsg.Cid: %v", notfoundmsg, notfoundmsg.Cid())
			return false
		}
	}

	for _, smsg := range smsgs {
		if message, ok := messageResultsMap[smsg.Cid().String()]; ok {
			if !equalForMessage(smsg.Message, message) {
				log.Errorf("compare smsgs failed, not equal! smsg: %+v, message: %+v", smsg.Message, message)
				return false
			}
		} else {
			log.Warnf("compare smsgs failed, message not found! smsg: %+v, smsg.Cid: %v", smsg.Message, smsg.Cid())
			// 兼容旧库，比较unsigedcid
			if message, ok := messageResultsMap[smsg.Message.Cid().String()]; ok {
				if !equalForMessage(smsg.Message, message) {
					log.Errorf("compare smsgs by unsignedcid failed, not equal! smsg: %+v, message: %+v", smsg.Message, message)
					return false
				}
			} else {
				notFoundSignedMessages = append(notFoundSignedMessages, *smsg)
			}
		}
	}

	for _, notfoundsignedmsg := range notFoundSignedMessages {
		found := false
		for _, message := range messageResultsMap {
			if message.From == strings.TrimPrefix(notfoundsignedmsg.Message.From.String(), string(notfoundsignedmsg.Message.From.String()[0])) && message.Nonce == notfoundsignedmsg.Message.Nonce {
				found = true
				break
			}
		}

		if !found {
			log.Errorf("compare smsgs failed, message not found! smsg: %+v, smsg.Message.Cid: %v, smsg.Cid: %v", notfoundsignedmsg.Message, notfoundsignedmsg.Message.Cid(), notfoundsignedmsg.Cid())
			return false
		}
	}

	return true
}

func equalForMessage(bmsg types.Message, message Message) bool {
	if bmsg.Version != message.Version {
		log.Error("Version of msg not equal!")
		return false
	}
	if strings.TrimPrefix(bmsg.To.String(), string(bmsg.To.String()[0])) != message.To {
		log.Error("To of msg not equal!")
		return false
	}
	if strings.TrimPrefix(bmsg.From.String(), string(bmsg.From.String()[0])) != message.From {
		log.Error("From of msg not equal!")
		return false
	}
	if bmsg.Nonce != message.Nonce {
		log.Error("Nonce of msg not equal!")
		return false
	}

	fil := new(big.Int)
	value, _ := fil.SetString(message.Value, 10)
	if bmsg.Value.Int.Cmp(value) != 0 {
		log.Error("Value of msg not equal!")
		return false
	}
	if bmsg.GasLimit != message.GasLimit {
		log.Error("GasLimit of msg not equal!")
		return false
	}
	gasFeeCap, _ := fil.SetString(message.GasFeeCap, 10)
	if bmsg.GasFeeCap.Int.Cmp(gasFeeCap) != 0 {
		log.Error("GasFeeCap of msg not equal!")
		return false
	}
	gasPremium, _ := fil.SetString(message.GasPremium, 10)
	if bmsg.GasPremium.Int.Cmp(gasPremium) != 0 {
		log.Error("GasPremium of msg not equal!")
		return false
	}
	if uint64(bmsg.Method) != message.Method {
		log.Error("Method of msg not equal!")
		return false
	}
	if bytes.Compare(bmsg.Params, message.Params) != 0 {
		log.Error("Params of msg not equal!")
		return false
	}

	return true
}
