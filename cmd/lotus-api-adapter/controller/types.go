package controller

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/system"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func IsStoragePowerActor(c cid.Cid) bool {
	if c == builtin0.StoragePowerActorCodeID {
		return true
	}

	if c == builtin2.StoragePowerActorCodeID {
		return true
	}

	if c == builtin3.StoragePowerActorCodeID {
		return true
	}

	if c == builtin4.StoragePowerActorCodeID {
		return true
	}

	if c == builtin5.StoragePowerActorCodeID {
		return true
	}

	if c == builtin6.StoragePowerActorCodeID {
		return true
	}

	if c == builtin7.StoragePowerActorCodeID {
		return true
	}

	return false
}

func IsRewardActor(c cid.Cid) bool {
	if c == builtin0.RewardActorCodeID {
		return true
	}

	if c == builtin2.RewardActorCodeID {
		return true
	}

	if c == builtin3.RewardActorCodeID {
		return true
	}

	if c == builtin4.RewardActorCodeID {
		return true
	}

	if c == builtin5.RewardActorCodeID {
		return true
	}

	if c == builtin6.RewardActorCodeID {
		return true
	}

	if c == builtin7.RewardActorCodeID {
		return true
	}

	return false
}

func IsInitActor(c cid.Cid) bool {
	if c == builtin0.InitActorCodeID {
		return true
	}

	if c == builtin2.InitActorCodeID {
		return true
	}

	if c == builtin3.InitActorCodeID {
		return true
	}

	if c == builtin4.InitActorCodeID {
		return true
	}

	if c == builtin5.InitActorCodeID {
		return true
	}

	if c == builtin6.InitActorCodeID {
		return true
	}

	if c == builtin7.InitActorCodeID {
		return true
	}

	return false
}

func IsStorageMarketActor(c cid.Cid) bool {
	if c == builtin0.StorageMarketActorCodeID {
		return true
	}

	if c == builtin2.StorageMarketActorCodeID {
		return true
	}

	if c == builtin3.StorageMarketActorCodeID {
		return true
	}

	if c == builtin4.StorageMarketActorCodeID {
		return true
	}

	if c == builtin5.StorageMarketActorCodeID {
		return true
	}

	if c == builtin6.StorageMarketActorCodeID {
		return true
	}

	if c == builtin7.StorageMarketActorCodeID {
		return true
	}

	return false
}

func IsVerifiedRegistryActor(c cid.Cid) bool {
	if c == builtin0.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin2.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin3.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin4.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin5.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin6.VerifiedRegistryActorCodeID {
		return true
	}

	if c == builtin7.VerifiedRegistryActorCodeID {
		return true
	}

	return false
}

func IsSystemActor(c cid.Cid) bool {
	if c == builtin0.SystemActorCodeID {
		return true
	}

	if c == builtin2.SystemActorCodeID {
		return true
	}

	if c == builtin3.SystemActorCodeID {
		return true
	}

	if c == builtin4.SystemActorCodeID {
		return true
	}

	if c == builtin5.SystemActorCodeID {
		return true
	}

	if c == builtin6.SystemActorCodeID {
		return true
	}

	if c == builtin7.SystemActorCodeID {
		return true
	}

	return false
}

func IsBurntFundsActor(addr address.Address) bool {
	if addr == builtin0.BurntFundsActorAddr {
		return true
	}

	if addr == builtin2.BurntFundsActorAddr {
		return true
	}

	if addr == builtin3.BurntFundsActorAddr {
		return true
	}

	if addr == builtin4.BurntFundsActorAddr {
		return true
	}

	if addr == builtin5.BurntFundsActorAddr {
		return true
	}

	if addr == builtin6.BurntFundsActorAddr {
		return true
	}

	if addr == builtin7.BurntFundsActorAddr {
		return true
	}

	return false
}

func MakeSystemState(store adt.Store, c cid.Cid) (system.State, error) {
	if c == builtin0.SystemActorCodeID {
		return system.MakeState(store, actors.Version0)
	}

	if c == builtin2.SystemActorCodeID {
		return system.MakeState(store, actors.Version2)
	}

	if c == builtin3.SystemActorCodeID {
		return system.MakeState(store, actors.Version3)
	}

	if c == builtin4.SystemActorCodeID {
		return system.MakeState(store, actors.Version4)
	}

	if c == builtin5.SystemActorCodeID {
		return system.MakeState(store, actors.Version5)
	}

	if c == builtin6.SystemActorCodeID {
		return system.MakeState(store, actors.Version6)
	}

	if c == builtin7.SystemActorCodeID {
		return system.MakeState(store, actors.Version7)
	}

	return nil, xerrors.Errorf("not system actor code: %v", c)
}
