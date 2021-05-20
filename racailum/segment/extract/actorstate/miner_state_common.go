package actorstate

import (
	"fmt"

	"github.com/ipfs/go-cid"

	"context"

	"github.com/filecoin-project/go-bitfield"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"

	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	adt4 "github.com/filecoin-project/specs-actors/v4/actors/util/adt"

	bstore "github.com/filecoin-project/lotus/blockstore"
	cstore "github.com/filecoin-project/lotus/chain/store"
)

func init() {
	empty2, err := newEmptyMinerStateV2()
	if err != nil {
		panic(fmt.Errorf("construct empty miner state v2: %w", err))
	}

	emptyMinerStateV2 = empty2

	empty3, err := newEmptyMinerStateV3()
	if err != nil {
		panic(fmt.Errorf("construct empty miner state v3: %w", err))
	}

	emptyMinerStateV3 = empty3

	empty4, err := newEmptyMinerStateV4()
	if err != nil {
		panic(fmt.Errorf("construct empty miner state v4: %w", err))
	}

	emptyMinerStateV4 = empty4
}

var (
	emptyMinerStateV2 *miner2.State
	emptyMinerStateV3 *miner3.State
	emptyMinerStateV4 *miner4.State
)

func isEmptyMinerStateV2(mst *miner2.State) bool {
	earlyCount, err := mst.EarlyTerminations.Count()
	if err != nil || earlyCount != 0 {
		return false
	}

	return isEmptyOrZero(mst.PreCommitDeposits) &&
		isEmptyOrZero(mst.LockedFunds) &&
		isEmptyOrZero(mst.FeeDebt) &&
		mst.VestingFunds.Equals(emptyMinerStateV2.VestingFunds) &&
		isEmptyOrZero(mst.InitialPledge) &&
		mst.PreCommittedSectors.Equals(emptyMinerStateV2.PreCommittedSectors) &&
		mst.PreCommittedSectorsExpiry.Equals(emptyMinerStateV2.PreCommittedSectorsExpiry) &&
		mst.AllocatedSectors.Equals(emptyMinerStateV2.AllocatedSectors) &&
		mst.Sectors.Equals(emptyMinerStateV2.Sectors) &&
		mst.Deadlines.Equals(emptyMinerStateV2.Deadlines)
}

// see https://github.com/filecoin-project/specs-actors/blob/v2.3.4/actors/builtin/miner/miner_actor.go#L96-L156
func newEmptyMinerStateV2() (*miner2.State, error) {
	ctx := context.Background()

	inMemStore := bstore.NewMemorySync()
	adtStore := adt2.WrapStore(ctx, cstore.ActorStore(ctx, inMemStore))

	emptyMap, err := adt2.MakeEmptyMap(adtStore).Root()
	if err != nil {
		return nil, err
	}

	emptyArray, err := adt2.MakeEmptyArray(adtStore).Root()
	if err != nil {
		return nil, err
	}

	emptyBitfield := bitfield.NewFromSet(nil)
	emptyBitfieldCid, err := adtStore.Put(ctx, emptyBitfield)
	if err != nil {
		return nil, err
	}

	emptyDeadline := miner2.ConstructDeadline(emptyArray)
	emptyDeadlineCid, err := adtStore.Put(ctx, emptyDeadline)
	if err != nil {
		return nil, err
	}

	emptyDeadlines := miner2.ConstructDeadlines(emptyDeadlineCid)
	emptyVestingFunds := miner2.ConstructVestingFunds()
	emptyDeadlinesCid, err := adtStore.Put(ctx, emptyDeadlines)
	if err != nil {
		return nil, err
	}

	emptyVestingFundsCid, err := adtStore.Put(ctx, emptyVestingFunds)
	if err != nil {
		return nil, err
	}

	state, err := miner2.ConstructState(cid.Undef, 0, 0, emptyBitfieldCid, emptyArray, emptyMap, emptyDeadlinesCid, emptyVestingFundsCid)
	return state, err
}

func isEmptyMinerStateV3(mst *miner3.State) bool {
	earlyCount, err := mst.EarlyTerminations.Count()
	if err != nil || earlyCount != 0 {
		return false
	}

	return isEmptyOrZero(mst.PreCommitDeposits) &&
		isEmptyOrZero(mst.LockedFunds) &&
		isEmptyOrZero(mst.FeeDebt) &&
		mst.VestingFunds.Equals(emptyMinerStateV3.VestingFunds) &&
		isEmptyOrZero(mst.InitialPledge) &&
		mst.PreCommittedSectors.Equals(emptyMinerStateV3.PreCommittedSectors) &&
		mst.PreCommittedSectorsExpiry.Equals(emptyMinerStateV3.PreCommittedSectorsExpiry) &&
		mst.AllocatedSectors.Equals(emptyMinerStateV3.AllocatedSectors) &&
		mst.Sectors.Equals(emptyMinerStateV3.Sectors) &&
		mst.Deadlines.Equals(emptyMinerStateV3.Deadlines)
}

// see https://github.com/filecoin-project/specs-actors/blob/v3.0.3/actors/builtin/miner/miner_state.go#L173-L230
func newEmptyMinerStateV3() (*miner3.State, error) {
	ctx := context.Background()
	inMemStore := bstore.NewMemorySync()
	adtStore := adt3.WrapStore(ctx, cstore.ActorStore(ctx, inMemStore))
	return miner3.ConstructState(adtStore, cid.Undef, 0, 0)
}

func isEmptyMinerStateV4(mst *miner4.State) bool {
	earlyCount, err := mst.EarlyTerminations.Count()
	if err != nil || earlyCount != 0 {
		return false
	}

	return isEmptyOrZero(mst.PreCommitDeposits) &&
		isEmptyOrZero(mst.LockedFunds) &&
		isEmptyOrZero(mst.FeeDebt) &&
		mst.VestingFunds.Equals(emptyMinerStateV4.VestingFunds) &&
		isEmptyOrZero(mst.InitialPledge) &&
		mst.PreCommittedSectors.Equals(emptyMinerStateV4.PreCommittedSectors) &&
		mst.PreCommittedSectorsExpiry.Equals(emptyMinerStateV4.PreCommittedSectorsExpiry) &&
		mst.AllocatedSectors.Equals(emptyMinerStateV4.AllocatedSectors) &&
		mst.Sectors.Equals(emptyMinerStateV4.Sectors) &&
		mst.Deadlines.Equals(emptyMinerStateV4.Deadlines)
}

// see https://github.com/filecoin-project/specs-actors/blob/v3.0.3/actors/builtin/miner/miner_state.go#L173-L230
func newEmptyMinerStateV4() (*miner4.State, error) {
	ctx := context.Background()
	inMemStore := bstore.NewMemorySync()
	adtStore := adt4.WrapStore(ctx, cstore.ActorStore(ctx, inMemStore))
	return miner4.ConstructState(adtStore, cid.Undef, 0, 0)
}
