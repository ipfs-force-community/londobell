package actorstate

import (
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"

	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"

	"github.com/dtynn/londobell/racailum/segment/model"
	"github.com/dtynn/londobell/racailum/segment/model/schema"
)

func inti() {
	schema.Register(
		schema.Model{
			Name: "actor-state: miner.State v2",
			D: &model.ActorState{
				Detail: &miner2.State{},
			},
		},

		schema.Model{
			Name: "actor-state: miner.State v3",
			D: &model.ActorState{
				Detail: &miner3.State{},
			},
		},

		schema.Model{
			Name: "actor-state: miner.State v4",
			D: &model.ActorState{
				Detail: &miner4.State{},
			},
		},
	)
}

func isEmptyState(st interface{}) bool {
	if st == nil {
		return true
	}

	switch st := st.(type) {
	case *miner2.State:
		return isEmptyMinerStateV2(st)

	case *miner3.State:
		return isEmptyMinerStateV3(st)

	case *miner4.State:
		return isEmptyMinerStateV4(st)

	case *miner5.State:
		return isEmptyMinerStateV5(st)
	default:
		return false
	}
}
