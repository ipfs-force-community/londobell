/*
--------------------------------------------------------------
@File    :   set_head.go
@Time    :   2023/11/22 11:24:52
@Author  :   lsk
@Version :   1.0
@Desc    :
--------------------------------------------------------------
删除正式库中的数据至设定的Epoch高度
--------------------------------------------------------------
*/
package main

import (
	"context"

	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dealProposalCollections = []string{"DealProposalDetail", "DealProposalSummary"}
	// messageCollection         = "Message"
	idCollections    = []string{"FinalHeight", "FilSupply", "StateFinalHeight", "Tipset"}
	epochCollections = []string{"ActorBalance", "ActorEvent", "MinerFunds", "ActorState", "MinerSector", "NewDealProposal", "ActorAddress", "CreateMessage", "MinerDealSector", "MultisigBalance", "BlockHeader", "MinerSectorSummary", "DealProposal", "PendingTxns", "EthHash", "BlockMessage", "ChangedActor", "MinerSectorHealth", "MarketFunds", "EvmInitCode", "SectorClaim", "ActorMessage", "EventsRoot", "ExecTrace", "VerifiedRegistry", "ClaimedPower", "MiningProfitability"}

	setHeadCmd = &cli.Command{
		Name:  "sethead",
		Usage: "sethead of bell formal db",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dsn",
				Required: true,
				Usage:    "dsn of database",
			},
			&cli.StringFlag{
				Name:     "name",
				Required: true,
				Usage:    "name of database",
			},
			&cli.Int64Flag{
				Name:     "epoch",
				Required: true,
				Usage:    "head of epoch",
			},
		},
		Action: func(cctx *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dsn := cctx.String("dsn")
			epoch := cctx.Int64("epoch")
			log.Infof("sethead,dsn: %s,epoch %d", dsn, epoch)
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
			if err != nil {
				log.Error(err)
				return err
			}

			db := client.Database(cctx.String("name"))
			err = setHead(ctx, db, epoch)
			if err != nil {
				log.Error("sethead err: ", err)
			} else {
				log.Infof("sethead %d succ", epoch)
			}

			return err
		},
	}
)

func setHead(ctx context.Context, db *mongo.Database, epoch int64) error {
	// message 表与Epoch关联,必须先删除
	err := deleteMessageDataByCid(ctx, db, epoch)
	if err != nil {
		return err
	}
	err = delEpochCols(ctx, db, epochCollections, epoch)
	if err != nil {
		return err
	}
	err = delIDCols(ctx, db, idCollections, epoch)
	if err != nil {
		return err
	}
	err = delPCols(ctx, db, dealProposalCollections, epoch)
	return err

}

// 删除epoch为Epoch字段的表数据
func delEpochCols(ctx context.Context, db *mongo.Database, collections []string, epoch int64) error {
	log.Info("deleteDataByEpoch start")
	defer log.Info("deleteDataByEpoch done")
	for _, collection := range collections {
		col := db.Collection(collection)
		_, err := col.DeleteMany(ctx, bson.M{"Epoch": bson.M{"$gt": epoch}})
		if err != nil {
			log.Errorf("update %s error: %s", collection, err)
			return err
		}
		log.Info(collection, " delete success")
	}

	return nil
}

// 删除epoch为_id字段的表数据
func delIDCols(ctx context.Context, db *mongo.Database, collections []string, epoch int64) error {
	log.Info("deleteDataByID start")
	defer log.Info("deleteDataByID done")
	for _, collection := range collections {
		col := db.Collection(collection)
		_, err := col.DeleteMany(ctx, bson.M{"_id": bson.M{"$gt": epoch}})
		if err != nil {
			log.Errorf("update %s error: %s", collection, err)
			return err
		}
		log.Info(collection, " delete success")
	}

	return nil
}

// 删除 dealProposalCollections 中的数据
func delPCols(ctx context.Context, db *mongo.Database, collections []string, epoch int64) error {
	log.Info("deleteDealProposalPre start")
	defer log.Info("deleteDealProposalPre done")
	for _, collection := range collections {
		col := db.Collection(collection)
		_, err := col.DeleteMany(ctx, bson.M{"ActorStateExBasic.Epoch": bson.M{"$gt": epoch}})
		if err != nil {
			log.Errorf("update %s error: %s", collection, err)
			return err
		}
		log.Info(collection, " delete success")
	}

	return nil
}

// 删除Message中的数据
func deleteMessageDataByCid(ctx context.Context, db *mongo.Database, epoch int64) error {
	log.Info("deleteMessageDataByCid start")
	defer log.Info("deleteMessageDataByCid done")
	execTraceCollection := db.Collection("ExecTrace")
	messageCollection := db.Collection("Message")
	cursor, err := execTraceCollection.Find(ctx, bson.M{"Epoch": bson.M{"$gt": epoch}}, options.Find().SetProjection(bson.M{"Cid": 1}))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var traceData bson.M
		if err := cursor.Decode(&traceData); err != nil {
			log.Error("get trace data error: ", err)
			return err
		}

		cid, ok := traceData["Cid"]
		if !ok {
			continue
		}

		_, err := messageCollection.DeleteOne(ctx, bson.M{"_id": cid})
		if err != nil {
			log.Error("delete message data error: ", err)
			return err
		}
		log.Info("del message : ", cid)
	}

	return nil
}
