package main

import (
	"context"

	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var fixFILCmd = &cli.Command{
	Name:  "fix-fil",
	Usage: "fix-fil",
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
		log.Infof("fix fil,dsn: %s", dsn)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
		if err != nil {
			log.Error(err)
			return err
		}

		db := client.Database(cctx.String("name"))
		col := db.Collection("ExecTrace")
		updateFIL(ctx, col)
		return nil
	},
}

func fixFIL(ctx context.Context, col *mongo.Collection) {

	filter := bson.M{
		"$or": []bson.M{
			{"FIL": bson.M{"$type": "double", "$eq": 0.0}},
			{"FIL": bson.M{"$type": "int", "$eq": 0}},
		},
	}

	// 查询需要更新的文档
	cur, err := col.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)
	current := 0
	for cur.Next(ctx) {
		var doc Document
		if current%10000 == 0 {
			log.Infof(" current field : %d", current)
		}
		err := cur.Decode(&doc)
		if err != nil {
			log.Fatal(err)
		}

		// 更新文档中的 "FIL" 字段
		_, err = col.UpdateOne(
			ctx,
			bson.M{"_id": doc.ID}, // 根据文档的唯一标识符进行更新
			bson.M{"$set": bson.M{"FIL": int64(0)}},
		)
		if err != nil {
			log.Error(err)
		}
		current++
	}
	log.Infof("fix fil done,%d field updated", current)
}
