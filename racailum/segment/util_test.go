package segment

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/ipfs-force-community/londobell/lib/mgoutil"

	"github.com/ipfs/go-cid"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
)

func TestExtractUpdateDoc(t *testing.T) {
	ctx := context.TODO()
	dsn := "mongodb://127.0.0.1:27017/test"
	wcli, err := mgoutil.Connect(context.TODO(), dsn)
	require.NoError(t, err)
	wdb := wcli.Database("test")

	scol := wdb.Collection("SectorClaim")

	d, _ := cid.Decode("baga6ea4seaqivfke4wzkazxtldtudcvofb4mgnlhyron6zhcqysjkslnrozhqii")
	c := model.NewSectorClaim(5359, abi.ActorID(1038), abi.ActorID(19538), d, 34359738368, 864000, 1555200, 727578, 153, 7227578)
	primaryKeyValue, err := ExtractPrimaryKeyValue(c)
	require.NoError(t, err)
	epochValue, err := ExtractEpochValue(c)
	require.NoError(t, err)

	updateDoc, err := ExtractUpdateDoc(c)
	require.NoError(t, err)

	var existingDoc bson.M
	err = scol.FindOne(ctx, bson.M{"_id": primaryKeyValue}).Decode(&existingDoc)
	if err != nil && err != mongo.ErrNoDocuments {
		return
	}

	var res = &mongo.UpdateResult{}
	if err == mongo.ErrNoDocuments {
		res, err = scol.UpdateMany(ctx, bson.M{"_id": primaryKeyValue}, bson.M{"$set": updateDoc}, options.Update().SetUpsert(true))
		require.NoError(t, err)
	} else {
		existingEpoch, ok := existingDoc["Epoch"].(int64)
		if !ok {
			return
		}

		if epochValue >= existingEpoch {
			res, err = scol.UpdateMany(ctx, bson.M{"_id": primaryKeyValue}, bson.M{"$set": updateDoc}, options.Update().SetUpsert(true))
			require.NoError(t, err)
		} else {
			fmt.Println("skip")
			return
		}
	}

	//mcol := wdb.Collection("MinerSector")
	//a, _ := address.NewFromString("t016765")
	//m := model.NewMinerSector(a, 1551, []abi.DealID{112713}, 727309, 1317699, big.NewInt(0), big.NewInt(0), false, abi.NewTokenAmount(1), false, 727309)
	//res, err := ExtractUpdateDoc(m)
	//require.NoError(t, err)

	fmt.Println(updateDoc)
	fmt.Println(res)
}
