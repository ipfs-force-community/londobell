package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

func TestBalance(t *testing.T) {
	addrs := make([]string, 0)
	f, _ := os.Open("/Users/zhoulin/zhoulin/test.txt")
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		a, _, c := r.ReadLine()
		if c == io.EOF {
			break
		}

		addrs = append(addrs, string(a))
	}

	js := "[\n    {\n        $match: {\n            \"Addr\": ctx.Addr,\n            \"Epoch\": 2830320\n        }\n    }\n]"
	var ctx context.Context
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://guest:read-only@dds-uf655172d52c38641732-pub.mongodb.rds.aliyuncs.com:3717/bell?replicaSet=mgset-65444697"))
	require.NoError(t, err, "failed")
	db := client.Database("bell")
	col := db.Collection("ActorBalance")

	notfound := make([]string, 0)
	file, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/notfound.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	require.NoError(t, err, "failed")
	defer file.Close()

	for _, addr := range addrs {
		addr = addr[1:]
		pipe, err := util.Parse(model.Ctx{Addr: addr}, string(js))
		require.NoError(t, err, "failed")

		var res []bson.M

		cur, err := col.Aggregate(ctx, pipe)
		require.NoError(t, err, "failed")
		err = cur.All(ctx, &res)
		require.NoError(t, err, "failed")

		if len(res) == 0 {
			io.WriteString(file, addr)
			io.WriteString(file, "\n")

			notfound = append(notfound, addr)
		}
	}

	fmt.Printf("notfound: %v\n", len(notfound))

}

func ReadLine2(filename string, addrs *[]string) error {
	f, _ := os.Open(filename)
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		err := readLine(r, addrs)
		if err != nil {
			return err
		}
	}

	return nil
}

func readLine(r *bufio.Reader, addrs *[]string) error {
	line, _, err := r.ReadLine()
	for err == nil {
		_, _, err = r.ReadLine()
		*addrs = append(*addrs, string(line))
	}
	return err
}
