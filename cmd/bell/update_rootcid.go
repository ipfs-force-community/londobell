/*
--------------------------------------------------------------
@File    :   update_rootcid.go
@Time    :   2023/10/11 11:17:48
@Author  :   lsk
@Version :   1.0
@Desc    :
--------------------------------------------------------------
为ActorMessage CreateMessage ExecTrace三张表添加rootcid信息
--------------------------------------------------------------
*/
package main

import (
	"context"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var updateRootCidCmd = &cli.Command{
	Name:  "update-rootcid",
	Usage: "update-rootcid",
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
	},
	Action: func(cctx *cli.Context) error {
		// set env
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dsn := cctx.String("dsn")
		log.Infof("update rootcid,dsn: %s", dsn)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
		if err != nil {
			log.Error(err)
			return err
		}
		colNames := []string{"ExecTrace", "ActorMessage", "CreateMessage"}
		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		for _, colName := range colNames {
			col := db.Collection(colName)
			updateRootCid(ctx, col, traceCol)
		}

		return nil
	},
}

// 检查字段是否存在
func FieldExist(ctx context.Context, col *mongo.Collection, fieldName string) bool {
	var err error

	filter := bson.M{
		fieldName: bson.M{"$exists": true},
	}

	var result bson.M
	err = col.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Infof("%s field %s dont exist", col.Name(), fieldName)
		} else {
			log.Error(err)
		}
		return false
	}
	return true

}

// getRootCids 从Exectrace表获取父cid
func getRootCids(ctx context.Context, col *mongo.Collection, childID string) (rootCid, rootSignedCid string, err error) {

	var root rootCidTrace

	rootID, err := model.GetRootID(childID)
	if err != nil {
		return rootCid, rootSignedCid, err
	}

	if rootID == childID {
		return rootCid, rootSignedCid, err
	}

	rootFilter := bson.M{"_id": rootID}
	err = col.FindOne(ctx, rootFilter).Decode(&root)
	if err == nil {
		if root.IsBlock {
			rootCid, rootSignedCid = root.Cid, root.SignedCid
		}
	}
	return rootCid, rootSignedCid, err

}

type rootCidTrace struct {
	ID            string `mir:"-" bson:"_id"`
	Cid           string
	SignedCid     string
	RootCid       string
	RootSignedCid string
	IsBlock       bool
}

func updateRootCid(ctx context.Context, baseCol, traceCol *mongo.Collection) {

	var filter bson.M
	// field存在表明部分字段已经更新
	if FieldExist(ctx, baseCol, "RootCid") {
		// 存在部分重复:父cid为系统消息时,RootCid为null,但是不影响结果
		filter = bson.M{"RootCid": bson.M{"$eq": nil}, "IsBlock": false}
	} else {
		filter = bson.M{"IsBlock": false}
	}
	// 查询需要更新的文档
	cur, err := baseCol.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)
	current := 0
	for cur.Next(ctx) {
		var doc rootCidTrace
		if current%10000 == 0 {
			log.Infof(" current field : %d", current)
		}
		err := cur.Decode(&doc)
		if err != nil {
			log.Fatal(err)
		}

		rootCid, rootSignedCid, err := getRootCids(ctx, traceCol, doc.ID)
		if err != nil {
			log.Warn(err)
		} else {
			if rootCid != "" {
				log.Debugf(" update field : %s", doc.ID)
				_, err = baseCol.UpdateOne(
					ctx,
					bson.M{"_id": doc.ID},
					bson.M{"$set": bson.M{"RootCid": rootCid, "RootSignedCid": rootSignedCid}},
				)
				if err != nil {
					log.Error(err)
				}
			}
		}
		current++
	}
	log.Infof("%s update rootcid done,%d field updated", baseCol.Name(), current)
}
