/*
--------------------------------------------------------------
@File    :   update_finalheight.go
@Time    :   2023/11/24 10:48:28
@Author  :   lsk
@Version :   1.0
@Desc    :
--------------------------------------------------------------
更新冷库中的FinalHeight表
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

var updateFinalHeight = &cli.Command{
	Name:  "update-finalheight",
	Usage: "update finalheight for cool db",
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dsn := cctx.String("dsn")
		dbName := cctx.String("name")
		log.Infof("update finalheight,dsn: %s,db: %s", dsn, dbName)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
		if err != nil {
			log.Error(err)
			return err
		}

		db := client.Database(dbName)

		err = UpdateFinalHeight(ctx, db)
		if err != nil {
			return err
		}
		log.Infof("update finalheight success,dsn: %s,db: %s", dsn, dbName)
		return nil
	},
}

func UpdateFinalHeight(ctx context.Context, db *mongo.Database) error {
	tipsetCol := db.Collection("Tipset")
	finalHeightCol := db.Collection("FinalHeight")

	var tipset model.FinalHeight

	tipsetSort := options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})
	err := tipsetCol.FindOne(ctx, bson.D{}, tipsetSort).Decode(&tipset)
	if err != nil {
		return err
	}

	_, err = finalHeightCol.InsertOne(ctx, tipset)
	return err
}
