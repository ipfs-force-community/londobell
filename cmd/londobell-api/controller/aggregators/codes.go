package aggregators

import (
	"sort"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// The built-in actor code IDs
var (
	SystemActorCodeID           cid.Cid
	InitActorCodeID             cid.Cid
	CronActorCodeID             cid.Cid
	AccountActorCodeID          cid.Cid
	StoragePowerActorCodeID     cid.Cid
	StorageMinerActorCodeID     cid.Cid
	StorageMarketActorCodeID    cid.Cid
	PaymentChannelActorCodeID   cid.Cid
	MultisigActorCodeID         cid.Cid
	RewardActorCodeID           cid.Cid
	VerifiedRegistryActorCodeID cid.Cid
	CallerTypesSignable         []cid.Cid
)

var builtinActors map[cid.Cid]*actorInfo

type actorInfo struct {
	name   string
	signer bool
}

func init() {
	builder := cid.V1Builder{Codec: cid.Raw, MhType: mh.IDENTITY}
	builtinActors = make(map[cid.Cid]*actorInfo)

	for id, info := range map[*cid.Cid]*actorInfo{ //nolint:nomaprange
		&SystemActorCodeID:           {name: "fil/1/system"},
		&InitActorCodeID:             {name: "fil/1/init"},
		&CronActorCodeID:             {name: "fil/1/cron"},
		&StoragePowerActorCodeID:     {name: "fil/1/storagepower"},
		&StorageMinerActorCodeID:     {name: "fil/1/storageminer"},
		&StorageMarketActorCodeID:    {name: "fil/1/storagemarket"},
		&PaymentChannelActorCodeID:   {name: "fil/1/paymentchannel"},
		&RewardActorCodeID:           {name: "fil/1/reward"},
		&VerifiedRegistryActorCodeID: {name: "fil/1/verifiedregistry"},
		&AccountActorCodeID:          {name: "fil/1/account", signer: true},
		&MultisigActorCodeID:         {name: "fil/1/multisig", signer: true},
	} {
		c, err := builder.Sum([]byte(info.name))
		if err != nil {
			panic(err)
		}
		*id = c
		builtinActors[c] = info
	}

	// Set of actor code types that can represent external signing parties.
	for id, info := range builtinActors { //nolint:nomaprange
		if info.signer {
			CallerTypesSignable = append(CallerTypesSignable, id)
		}
	}
	sort.Slice(CallerTypesSignable, func(i, j int) bool {
		return CallerTypesSignable[i].KeyString() < CallerTypesSignable[j].KeyString()
	})

}
