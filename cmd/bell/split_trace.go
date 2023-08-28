/*
--------------------------------------------------------------
@File    :   split_trace.go
@Time    :   2023/08/23 13:57:37
@Author  :   lsk
@Version :   1.0
@Desc    :

	根据methodName将execTrace表进行切割
	PS: 当前只支持一次性的全量操作,后续该功能会合并入londonbell
	Run Example:
	./bell split-trace -dsn mongodb://localhost:27017 -name test

--------------------------------------------------------------
*/
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	apim "github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TraceMsg struct {
	ID        string `bson:"_id" json:"_id"`
	Epoch     abi.ChainEpoch
	Cid       string
	SignedCid string
	Msg       struct {
		From       string
		To         string
		Method     uint64
		Value      string
		MethodName string
	}
	MsgRct struct {
		version    byte
		ExitCode   int64
		Return     []byte
		GasUsed    int64
		EventsRoot string
	}
	Detail struct {
		Return interface{}
	}
	IsBlock bool
}

// Trace2Message Convert ExecTrace to MessageForCreate
func Trace2Message(trace TraceMsg, col *mongo.Collection) (msg apim.MessageForCreate) {
	msg.Cid = trace.Cid
	msg.Epoch = trace.Epoch
	// msg.ExitCode
	msg.From = trace.Msg.From
	msg.To = trace.Msg.To
	msg.Method = trace.Msg.MethodName
	msg.ActorID = getActorID(trace)
	msg.Value = trace.Msg.Value
	msg.ID = trace.ID
	msg.Caller = getCaller(trace.Msg.MethodName, trace.ID, col)
	return
}

// getCaller get contructor caller address
func getCaller(methodName, id string, col *mongo.Collection) string {
	if methodName == model.ConstructorMethod {
		var execTrace TraceMsg
		var callerID string
		parts := strings.Split(id, "-")

		// 取前两个分割后的部分
		if len(parts) >= 2 {
			callerID = parts[0] + "-" + parts[1]
		} else {
			log.Error("Parse caller id error,cid : ", id)
			return ""
		}
		err := col.FindOne(context.TODO(), bson.D{{Key: "_id", Value: callerID}}).Decode(&execTrace)
		if err != nil {
			log.Error(fmt.Sprintf("find caller msg err: %v,callerID: %s", err, callerID))
			return ""
		}
		return execTrace.Msg.From
	}

	return ""
}

func getActorID(trace TraceMsg) string {
	if trace.Msg.MethodName == model.CreateExternal {

		actorID := trace.Detail.Return.(bson.D).Map()["ActorID"].(int64)
		return fmt.Sprintf("0%d", actorID)
	}
	return ""

}

var splitTraceCmd = &cli.Command{
	Name:  "split-trace",
	Usage: "split traceExec collection",
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
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			log.Error(err)
			return err
		}

		db := client.Database(cctx.String("name"))
		TraceCol := db.Collection("ExecTrace")

		// read ExecTrace table
		cursor, err := TraceCol.Find(context.Background(), bson.M{})
		if err != nil {
			log.Error(err)
			return err
		}
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {

			var execTrace TraceMsg

			if err := cursor.Decode(&execTrace); err != nil {
				log.Errorf(fmt.Sprintf("Decode: %s,Err: %v", cursor.Current.String(), err))
				return err
			}
			methodName := execTrace.Msg.MethodName
			//
			if model.IsOkCreateMessage(methodName, execTrace.MsgRct.ExitCode) {
				var cmsg model.CreateMessage
				newCollection := db.Collection(cmsg.CollectionName())
				msg := Trace2Message(execTrace, TraceCol)
				_, err = newCollection.InsertOne(context.Background(), msg)
				if err != nil {
					log.Errorf(fmt.Sprintf("Insert: %s,Err: %v", cursor.Current.String(), err))
				}
			}

		}
		return nil
	},
}
