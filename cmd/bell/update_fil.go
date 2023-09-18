/*
--------------------------------------------------------------
@File    :   update_fil.go
@Time    :   2023/09/14 17:44:56
@Author  :   lsk
@Version :   1.0
@Desc    :
--------------------------------------------------------------
根据现有的ExecTrace Msg.Value字段为其添加 FIL字段(int64)以支持大额转账
全量更新整张ExecTrace表
Example: ./bell update-fil -dsn mongodb://localhost:27017/ -name dbname
可使用mongo命令:
```
db.getCollection("ExecTrace").updateMany(

	{},
	[
	    {
	        $addFields: {
	            FIL: {
	                $cond: {
	                    if: { $gt: [{ $strLenCP: "$Msg.Value" }, 18] },
	                    then: {
	                        $toInt: {
	                            $substr: ["$Msg.Value", 0, { $subtract: [{ $strLenCP: "$Msg.Value" }, 18] }]
	                        }
	                    },
	                    else: 0
	                }
	            }
	        }
	    }
	]

)
```

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

var updateFILCmd = &cli.Command{
	Name:  "update-fil",
	Usage: "update-fil",
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

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cctx.String("dsn")))
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

type Document struct {
	ID string `bson:"_id"`
	Msg
	FIL int64
}
type Msg struct {
	Value string
}

func updateFIL(ctx context.Context, col *mongo.Collection) {
	// 设置MongoDB连接

	// 查询需要更新的文档
	cur, err := col.Find(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var doc Document
		err := cur.Decode(&doc)
		if err != nil {
			log.Fatal(err)
		}

		// 计算 "FIL" 字段的值
		filValue := model.CalculateFILValue(doc.Msg.Value)

		// 更新文档中的 "FIL" 字段
		_, err = col.UpdateOne(
			context.TODO(),
			bson.M{"_id": doc.ID}, // 根据文档的唯一标识符进行更新
			bson.M{"$set": bson.M{"FIL": filValue}},
		)
		if err != nil {
			log.Fatal(err)
		}
	}
}
