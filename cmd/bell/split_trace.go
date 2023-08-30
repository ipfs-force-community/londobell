/*
--------------------------------------------------------------
@File    :   split_trace.go
@Time    :   2023/08/23 13:57:37
@Author  :   lsk
@Version :   1.0
@Desc    :

	将execTrace表id倒序,根据(start,end)区间进行切割
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
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	apim "github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/urfave/cli/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SplitTask struct {
	ID     primitive.ObjectID `bson:"_id"`
	Start  string
	End    string
	Status bool
}

// ExecTrace collection data structure
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
		err := col.FindOne(context.TODO(), bson.M{"_id": callerID}).Decode(&execTrace)
		if err != nil {
			log.Error(fmt.Sprintf("find caller msg err: %v,callerID: %s", err, callerID))
			return ""
		}
		return execTrace.Msg.From
	}

	return ""
}

// get creat ActorID
func getActorID(trace TraceMsg) string {
	if trace.Msg.MethodName == model.CreateExternal {

		actorID := trace.Detail.Return.(bson.D).Map()["ActorID"].(int64)
		return fmt.Sprintf("0%d", actorID)
	} else if trace.Msg.MethodName == model.CreateMiner || trace.Msg.MethodName == model.Exec {
		if actorID, ok := trace.Detail.Return.(bson.D).Map()["IDAddress"].(string); ok {
			return actorID
		}
		return ""
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
		&cli.StringFlag{
			Name:     "start",
			Required: false,
			Usage:    "id of start",
		},
		&cli.StringFlag{
			Name:     "end",
			Required: false,
			Usage:    "id of end",
		},
		&cli.BoolFlag{
			Name:     "reRun",
			Required: false,
			Usage:    "reRun last failed tasks",
		},
	},
	Action: func(cctx *cli.Context) error {
		// set env
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var splitTask SplitTask
		var cmsg model.CreateMessage
		var reRun bool
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cctx.String("dsn")))
		if err != nil {
			log.Error(err)
			return err
		}

		db := client.Database(cctx.String("name"))
		traceCol := db.Collection("ExecTrace")
		taskCol := db.Collection("SplitTask")
		createCol := db.Collection(cmsg.CollectionName())

		splitTask.Start = cctx.String("start")
		splitTask.End = cctx.String("end")
		reRun = cctx.Bool("reRun")
		if reRun && (splitTask.Start+splitTask.End != "") {
			log.Fatal("reRun cannot be used together with start or end")
		}
		splitTask.Status = false
		tasks, err := initTasks(ctx, taskCol, traceCol, splitTask, reRun)
		if err != nil {
			log.Fatal(err)
		}
		for _, task := range tasks {
			err := task.run(ctx, taskCol, traceCol, createCol)
			if err != nil {
				log.Errorf("tasks exec error,tasks: %v,err: %w", tasks, err)
				return err
			}
		}
		return nil
	},
}

func initTasks(ctx context.Context, taskCol, traceCol *mongo.Collection, st SplitTask, reRun bool) ([]SplitTask, error) {
	var results []SplitTask
	var err error
	if reRun {
		cursor, err := taskCol.Find(ctx, bson.M{"Status": false})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		if err := cursor.All(ctx, &results); err != nil {
			log.Error(err)
			return nil, err
		}
	} else {
		// if start not set,use ExecTrace collection latest id
		if st.Start == "" {
			st.Start = getLatestTraceID(ctx, traceCol)
		}
		st.ID = primitive.NewObjectID()
		_, err = taskCol.InsertOne(ctx, st)
		if err != nil {
			log.Fatal(err)
		}
	}

	results = append(results, st)
	return results, nil
}

// get ExecTrace collection latest id
func getLatestTraceID(ctx context.Context, traceCol *mongo.Collection) string {
	var execTrace TraceMsg
	opts := options.FindOne()
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	// read ExecTrace table
	result := traceCol.FindOne(ctx, bson.M{}, opts)
	err := result.Decode(&execTrace)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("get latest trace id :", execTrace.ID)
	return execTrace.ID
}

func (sp SplitTask) updateStart(ctx context.Context, taskCol *mongo.Collection) error {
	update := bson.M{"$set": bson.M{"Start": sp.Start, "Status": sp.Status}}
	_, err := taskCol.UpdateByID(ctx, sp.ID, update)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (sp SplitTask) updateEnd(ctx context.Context, taskCol *mongo.Collection) error {
	update := bson.M{"$set": bson.M{"End": sp.End, "Status": sp.Status}}
	_, err := taskCol.UpdateByID(ctx, sp.ID, update)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (sp SplitTask) run(ctx context.Context, taskCol, TraceCol, createCol *mongo.Collection) error {

	var TotalCount, ProcessedCount, InsertCount int64
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})

	minID := sp.End   // 起始 ID
	maxID := sp.Start // 结束 ID
	log.Infof("split task start: %s,end: %s", sp.Start, sp.End)
	start := time.Now()
	// 构建查询过滤器
	filter := bson.M{"_id": bson.M{"$gte": minID, "$lte": maxID}}

	cursor, err := TraceCol.Find(ctx, filter, opts)
	if err != nil {
		log.Error(err)
		return err
	}
	defer cursor.Close(ctx)
	countOptions := options.Count().SetHint("_id_")

	if TotalCount, err = TraceCol.CountDocuments(ctx, filter, countOptions); err != nil {
		log.Error(err)
		return err
	}

	var execTrace TraceMsg

	for cursor.Next(ctx) {
		if err := cursor.Decode(&execTrace); err != nil {
			log.Errorf(fmt.Sprintf("Decode: %s,Err: %v", cursor.Current.String(), err))
			return err
		}
		methodName := execTrace.Msg.MethodName

		if model.IsOkCreateMessage(methodName, execTrace.MsgRct.ExitCode) {
			msg := Trace2Message(execTrace, TraceCol)
			_, err = createCol.InsertOne(ctx, msg)

			if err != nil {
				if writeErr, ok := err.(mongo.WriteException); ok {
					for _, we := range writeErr.WriteErrors {
						if we.Code == 11000 { // MongoDB错误码：11000 表示Duplicate Key Error
							log.Warn(we.Message)
						} else {
							log.Fatal(err)
						}
					}
				} else {
					log.Fatal(err)
				}
			} else {
				InsertCount++
			}
		}

		ProcessedCount++
	}
	sp.End = execTrace.ID
	if ProcessedCount == TotalCount {
		log.Infof("Success!!! Total %d fields,%d fileds processed,%d fileds inserted,all fileds processed", TotalCount, ProcessedCount, InsertCount)
		sp.Status = true
	} else {
		log.Infof("Failed!!! Total %d fields,%d fileds processed,%d fileds inserted", TotalCount, ProcessedCount, InsertCount)
		sp.Status = false
	}

	// update task end
	err = sp.updateEnd(ctx, taskCol)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("task done,spent %f's", time.Now().Sub(start).Seconds())
	return nil
}
