/*
--------------------------------------------------------------
@File    :   orphan_daemon.go
@Time    :   2023/10/19 13:55:16
@Author  :   lsk
@Version :   1.0
@Desc    :
--------------------------------------------------------------
同步lotus孤块数据:
https://github.com/filecoin-project/lotus/pull/632

index: db.OrphanBlock.createIndex({"Epoch":1}, {"sparse": true});
--------------------------------------------------------------
*/
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/ipfs-force-community/londobell/lib/mir"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	defaultCleanInterval int64 = 500
	// 节点第一次发现块时间
	b1sCache = expirable.NewLRU[cid.Cid, int64](10000, nil, time.Hour*1)
)

type OrphanBlock struct {
	ID            cid.Cid `mir:"-" bson:"_id" `
	Miner         address.Address
	Epoch         abi.ChainEpoch `mir:"Height"`
	Messages      cid.Cid
	ElectionProof *types.ElectionProof
	Ticket        *types.Ticket
	FirstSeen     int64
	Checked       bool
	MessageCount  int
	Parents       []cid.Cid
}

var orphanDaemonCmd = &cli.Command{
	Name:  "orphan-daemon",
	Usage: "get orphan block",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "dsn",
			Required: true,
			Usage:    "formal database dsn",
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Usage:    "name of database",
		},
		&cli.Int64Flag{
			Name:    "clean-interval",
			Aliases: []string{"ci"},
			Usage:   "interval block number of clean chain block ",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("orphan-daemon start")
		defer log.Info("orphan-daemon stop")

		ctx := context.Background()
		var requestHeader http.Header
		token := cctx.String("token")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		full, closer, err := client.NewFullNodeRPCV0(cctx.Context, cctx.String("api-url"), requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		client, err := mongo.Connect(cctx.Context, options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			return err
		}
		db := client.Database(cctx.String("name"))
		orphanCol := db.Collection("OrphanBlock")
		go updateBlockInfo(cctx, db)
	SYNCLOOP:
		sub, err := full.SyncIncomingBlocks(ctx)
		if err != nil {
			log.Errorf("subscribe incoming block err:%w", err)
			time.Sleep(15 * time.Second)
			goto SYNCLOOP
		}

		for bh := range sub {
			bCid := bh.Cid()
			var msgCount int
			bm, err := full.ChainGetBlockMessages(ctx, bCid)
			if err != nil {
				log.Errorf("ChainGetBlockMessages err:%w", err)
			} else {

				msgCount = len(bm.BlsMessages) + len(bm.SecpkMessages)
			}
			// bh, err := model.NewBlockHeader(bh.Miner, bh)
			if _, ok := b1sCache.Get(bCid); ok {
				continue
			}
			log.Infof("get block: %s", bCid)
			now := time.Now().Unix()
			ob := &OrphanBlock{
				ID:           bh.Cid(),
				FirstSeen:    now,
				MessageCount: msgCount,
			}
			b1sCache.Add(bCid, now)
			if err := mir.Mirror(ob, bh); err != nil {
				log.Errorf("mirroring OrphanBlock: %w", err)
			} else {
				err := saveOrphanBlock(ctx, orphanCol, ob)
				if err != nil {
					// TODO 数据库出错,存入leveldb?或其它嵌入式数据库
					log.Errorf("saveBlock failed: %w", err)
				}
			}

		}
		log.Warn("sub closed")
		goto SYNCLOOP

	},
}

func saveOrphanBlock(ctx context.Context, orphanCol *mongo.Collection, ob *OrphanBlock) error {
	_, err := orphanCol.InsertOne(ctx, ob)
	return err
}

// updateBlockInfo 更新OrphanBlock及BlockHeader
//
//	@param cctx
//	@param db
func updateBlockInfo(cctx *cli.Context, db *mongo.Database) {
	var (
		interval int64
		err      error
	)
	type finalHeightRes struct {
		Epoch int64 `bson:"_id"`
	}

	if !cctx.IsSet("clean-interval") {
		interval = defaultCleanInterval
	} else {
		interval = cctx.Int64("clean-interval")
	}

	orphanBlockCol := db.Collection("OrphanBlock")
	blockHeaderCol := db.Collection("BlockHeader")
	finalHeightCol := db.Collection("FinalHeight")
	for {
		// 等待interval个块,更新一次
		sleep := 30 * time.Duration(interval) * time.Second
		log.Infof("after %s updateBlockInfo ", sleep)
		time.Sleep(sleep)
		findOptions := options.FindOne()
		findOptions.SetSort(bson.D{{Key: "_id", Value: -1}})

		var result finalHeightRes
		err = finalHeightCol.FindOne(context.Background(), bson.M{}, findOptions).Decode(&result)
		if err != nil {
			log.Errorf("get finalHeight err: %w", err)
			continue
		}
		log.Infof("start updateBlockInfo endEpoch: %d", result.Epoch)
		// Step 1: 查询 OrphanBlock Checked 为 false 的数据,限定Epoch<=FinalHeight
		orphanBlockFilter := bson.M{
			"Checked": false,
			"Epoch":   bson.M{"$lte": result.Epoch},
		}
		orphanBlockCursor, err := orphanBlockCol.Find(context.Background(), orphanBlockFilter)
		if err != nil {
			log.Error("Error querying OrphanBlock:", err)
			continue
		}

		// Step 2: 遍历待CheckOrphanBlock
		for orphanBlockCursor.Next(context.Background()) {
			var orphanBlock OrphanBlock
			if err := orphanBlockCursor.Decode(&orphanBlock); err != nil {
				log.Error("Error decoding OrphanBlock:", err)
				continue
			}

			// Step 3: 根据 OrphanBlock ID 查询 BlockHeader
			IDFilter := bson.M{"_id": orphanBlock.ID}
			blockHeaderResult := blockHeaderCol.FindOne(context.Background(), IDFilter)

			if blockHeaderResult.Err() == mongo.ErrNoDocuments {
				// Step 4: 如果在 BlockHeader 中没有找到匹配的记录，说明为孤块,保留数据,更新Checked字段为true
				log.Infof("confirm OrphanBlock: %s", orphanBlock.ID)
				update := bson.M{"$set": bson.M{"Checked": true}}
				_, err := orphanBlockCol.UpdateOne(context.Background(), IDFilter, update)
				if err != nil {
					log.Error("Error updating OrphanBlock:", err)
					continue
				}
			} else if blockHeaderResult.Err() != nil {
				log.Error("Error querying BlockHeader:", blockHeaderResult.Err())
				continue
			} else {
				log.Infof("del chain block: %s,update blockHeader firstseen: %d", orphanBlock.ID, orphanBlock.FirstSeen)
				// Step 5: 如果在 BlockHeader 中找到匹配的记录更新BlockHeader FirstSeen字段，之后删除 OrphanBlock 数据
				update := bson.M{"$set": bson.M{"FirstSeen": orphanBlock.FirstSeen}}
				_, err := blockHeaderCol.UpdateOne(context.Background(), IDFilter, update)
				if err != nil {
					log.Error("Error updating BlockHeader:", err)
					continue
				}
				_, err = orphanBlockCol.DeleteOne(context.Background(), IDFilter)
				if err != nil {
					log.Errorf("Error deleting OrphanBlock:%w", err)
					continue
				}
			}
		}
	}

}
