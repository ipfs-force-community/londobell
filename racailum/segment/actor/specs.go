package actor

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/lotus/chain/actors"
	exported0 "github.com/filecoin-project/specs-actors/actors/builtin/exported"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	exported2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"

	exported3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/exported"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"

	exported4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/exported"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"

	exported5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/exported"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	exported6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/exported"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
)

func init() {
	if len(actors.Versions) != len(Specs) {
		panic(fmt.Errorf("length of specs & versions are not equal, %d != %d", len(Specs), len(actors.Versions)))
	}

	if len(actors.Versions) != len(WatchOuts) {
		panic(fmt.Errorf("length of watch-out types & versions are not equal, %d != %d", len(WatchOuts), len(actors.Versions)))
	}

	for i := 1; i < len(WatchOuts); i++ {
		if len(WatchOuts[i]) != len(WatchOuts[i-1]) {
			panic(fmt.Errorf("number of watch-out types are differet for %d & %d, %d != %d", i-1, i, len(WatchOuts[i-1]), len(WatchOuts[i])))
		}
	}
}

var Specs = [][]rt.VMActor{
	exported0.BuiltinActors(),
	exported2.BuiltinActors(),
	exported3.BuiltinActors(),
	exported4.BuiltinActors(),
	exported5.BuiltinActors(),
	exported6.BuiltinActors(),
}

var WatchOuts = [][]interface{}{
	[]interface{}{
		miner0.Deadline{},
		miner0.Partition{},
	},
	[]interface{}{
		miner2.Deadline{},
		miner2.Partition{},
	},
	[]interface{}{
		miner3.Deadline{},
		miner3.Partition{},
	},
	[]interface{}{
		miner4.Deadline{},
		miner4.Partition{},
	},
	[]interface{}{
		miner5.Deadline{},
		miner5.Partition{},
	},
	[]interface{}{
		miner6.Deadline{},
		miner6.Partition{},
	},
}
