package gen

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/testutils"
)

func TestExtractMinerFundsV7(t *testing.T) {
	ctx := context.Background()
	res := extract.NewRes(0, 0)
	localBs, closer, err := testutils.NewLocalBlockStore(ctx)
	require.NoError(t, err)
	defer func() {
		_ = closer()
	}()
	headerCId, _ := cid.Decode(testMinerSectorActorCid)
	addr, _ := address.NewFromString("f031582")
	actorStore := store.ActorStore(ctx, localBs)
	var out miner7.State
	err = actorStore.Get(ctx, headerCId, &out)
	require.NoError(t, err)
	mockDAL := &MockDAL{}
	mockDAL.On("ActorStore", ctx).Return(store.ActorStore(ctx, localBs), nil)
	latestDealID := int64(0)
	ectx, _ := extract.NewCtx(ctx, mockDAL, &zap.SugaredLogger{}, &actor.Set{}, latestDealID, extract.DryOptions())
	err = extractMinerFundsV7(ectx, res, &common.ActorHead{
		Actor: &types.Actor{Head: headerCId, Balance: big.MustFromString("2631953892560049294647")},
		Addr:  addr,
		Epoch: abi.ChainEpoch(792000),
	}, &out)

	require.NoError(t, err)
	require.Equal(t, 1, len(res.Docs))
	resultMoc := MinerFundsExpectResult()
	cpw := res.Docs[0].(*model.MinerFunds)
	require.Equal(t, cpw, resultMoc)

	//require.Equal(t, cpw.Info.State, resultMoc.Info.State)
	//require.Equal(t, cpw.Info.Owner, resultMoc.Info.Owner)
	//require.Equal(t, cpw.Info.Worker, resultMoc.Info.Worker)
	//require.Equal(t, cpw.Info.ControlAddresses, resultMoc.Info.ControlAddresses)
	//require.Equal(t, cpw.Info.PendingWorkerKey, resultMoc.Info.PendingWorkerKey)
	//require.Equal(t, cpw.Info.PeerID, resultMoc.Info.PeerID)
	//require.Equal(t, cpw.Info.Multiaddrs, resultMoc.Info.Multiaddrs)
	//require.Equal(t, cpw.Info.WindowPoStProofType, resultMoc.Info.WindowPoStProofType)
	//require.Equal(t, cpw.Info.SectorSize, resultMoc.Info.SectorSize)
	//require.Equal(t, cpw.Info.WindowPoStPartitionSectors, resultMoc.Info.WindowPoStPartitionSectors)
	//require.Equal(t, cpw.Info.ConsensusFaultElapsed, resultMoc.Info.ConsensusFaultElapsed)
	//require.Equal(t, cpw.Info.PendingOwnerAddress, resultMoc.Info.PendingOwnerAddress)
	//require.Equal(t, cpw.Info.Balance, resultMoc.Info.Balance)
	//require.Equal(t, cpw.Info.AvailableBalance, resultMoc.Info.AvailableBalance)
	//require.Equal(t, cpw.Info.FeeDebt, resultMoc.Info.FeeDebt)
	//require.Equal(t, cpw.Info.PrecommitSectorCount, resultMoc.Info.PrecommitSectorCount)
	//require.Equal(t, cpw.Info.State, resultMoc.Info.State)

	//require.Equal(t, cpw.ID, resultMoc.ID)
	//require.Equal(t, cpw.Addr, resultMoc.Addr)
	//require.Equal(t, cpw.Path, resultMoc.Path)
	//require.Equal(t, cpw.Epoch, resultMoc.Epoch)
}

func MinerFundsExpectResult() *model.MinerFunds {
	var ds = model.MinerFunds{}
	addr, _ := address.NewFromString("f031582")
	id, _ := cid.Decode("bafy2bzacea3kn2b3f5xedjfqsv4xdnveek47glbsqnzsuomzoiqo5lndmaitm")
	path1, _ := cid.Decode("bafy2bzacebtzupwrfw2fgmfqzafht4imbhbks2fkaqfs77ay2cc63au3jap3c")
	epoch := abi.ChainEpoch(792000)
	preCommitDeposits := big.MustFromString("252729872194318507828")
	lockedFunds := big.MustFromString("1417239072189471065603")
	feeDebt := abi.NewTokenAmount(0)
	initialPledge := big.MustFromString("197999996892736389120")
	vestInFuture := []abi.TokenAmount{
		big.MustFromString("8319219341634931532"),
		big.MustFromString("48916327422298198590"),
		big.MustFromString("57038855703069497329"),
		big.MustFromString("130374527321301708179"),
	}
	pledgeRelease := []abi.TokenAmount{
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
		abi.NewTokenAmount(0),
	}
	owner, _ := address.NewFromString("f031307")
	worker, _ := address.NewFromString("f031307")
	//controlAddresses: nil
	peerID := []byte{0, 36, 8, 1, 18, 32, 226, 51, 16, 88, 27, 182, 127, 111, 211, 173, 80, 201, 214, 72, 19, 227, 135, 234, 176, 165, 71, 1, 201, 27, 44, 93, 178, 199, 205, 205, 80, 61}
	multiaddrs := []abi.Multiaddrs{[]byte{4, 172, 20, 10, 203, 6, 37, 194}}
	windowPoStProofType := abi.RegisteredPoStProof(8)
	sectorSize := abi.SectorSize(34359738368)
	windowPoStPartitionSectors := uint64(2349)
	consensusFaultElapsed := abi.ChainEpoch(-1)
	//pendingOwnerAddress: nil
	balance := big.MustFromString("2631953892560049294647")
	availableBalance := big.MustFromString("763984951283523332096")
	feeDebt2 := abi.NewTokenAmount(0) // todo: 等于feeDebt？
	precommitSectorCount := uint64(11)

	info, _ := cid.Decode("bafy2bzacebrfty4brer6alic7y5k5xqsn75eh43xs5qukqn3dm3lz5x6stw6i")
	vestingFunds, _ := cid.Decode("bafy2bzacea5uz2oyo7cyylxqtswquzrn72m63y67wn364v7zcoxyctmr275o2")
	preCommittedSectors, _ := cid.Decode("bafy2bzaceaaluasbxetnykiwngn42reabg36eorplmg7op75qzsqzst3jmhlq")
	preCommittedSectorsCleanUp, _ := cid.Decode("bafy2bzacecmi2lvsyjrbsyz4vgk32ymjnoausal2ryhdn76ouztsivh62xbdo")
	allocatedSectors, _ := cid.Decode("bafy2bzaced5j4fk3cbe5zqvhd7fh6kxz5zhhogp5kax34ph55svscpmwetlye")
	sectors, _ := cid.Decode("bafy2bzacecn2msklq7kazx6u6khlgy6ocfm5pa2dtiwlya3oplqpip6gfdsfi")
	deadlines, _ := cid.Decode("bafy2bzacea2cnypfzf6pf2by35zwhqda6vpjh4cwckuosbkimr5mz4ndgemyu")
	state := miner7.State{
		Info:                       info,
		PreCommitDeposits:          big.MustFromString("252729872194318507828"),
		LockedFunds:                big.MustFromString("1417239072189471065603"),
		VestingFunds:               vestingFunds,
		FeeDebt:                    feeDebt2,
		InitialPledge:              big.MustFromString("197999996892736389120"),
		PreCommittedSectors:        preCommittedSectors,
		PreCommittedSectorsCleanUp: preCommittedSectorsCleanUp,
		AllocatedSectors:           allocatedSectors,
		Sectors:                    sectors,
		ProvingPeriodStart:         abi.ChainEpoch(791819),
		CurrentDeadline:            uint64(3),
		Deadlines:                  deadlines,
		EarlyTerminations:          bitfield.New(),
		DeadlineCronActive:         true,
	}
	ds = model.MinerFunds{
		ActorStateExBasic: model.ActorStateExBasic{ID: id, Path: []cid.Cid{path1}, Addr: addr, Epoch: epoch},
		Detail: model.MinerFundsDetail{
			PreCommitDeposits: preCommitDeposits,
			LockedFunds:       lockedFunds,
			FeeDebt:           feeDebt,
			InitialPledge:     initialPledge,
			VestInFuture:      vestInFuture,
			PledgeRelease:     pledgeRelease,
		},
		Info: model.MinerInfo{
			Owner:  owner,
			Worker: worker,
			//ControlAddresses:
			PendingWorkerKey:           model.WorkerKeyChange{EffectiveAt: abi.ChainEpoch(0)},
			PeerID:                     peerID,
			Multiaddrs:                 multiaddrs,
			WindowPoStProofType:        windowPoStProofType,
			SectorSize:                 sectorSize,
			WindowPoStPartitionSectors: windowPoStPartitionSectors,
			ConsensusFaultElapsed:      consensusFaultElapsed,
			//PendingOwnerAddress:
			Balance:              balance,
			AvailableBalance:     availableBalance,
			FeeDebt:              feeDebt2,
			PrecommitSectorCount: precommitSectorCount,
			State:                &state,
		},
	}
	return &ds
}
