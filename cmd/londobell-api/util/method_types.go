package util

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs/go-cid"
)

type ActorSet struct {
	m      map[address.Address]cid.Cid
	loadmu sync.RWMutex
}

var (
	UpgradeActorSet = make(map[abi.ChainEpoch]*ActorSet)
	lock            sync.RWMutex
	HeightSlice     []abi.ChainEpoch
)

func NewActorSet() *ActorSet {
	m := make(map[address.Address]cid.Cid)
	return &ActorSet{m: m}
}
func init() {
	var UpgradeInfinityHeight = abi.ChainEpoch(99999999999999)
	HeightSlice = append(HeightSlice, UpgradeInfinityHeight)
	UpgradeActorSet[UpgradeInfinityHeight] = NewActorSet()
	for _, u := range filcns.DefaultUpgradeSchedule() {
		UpgradeActorSet[u.Height] = NewActorSet()
		HeightSlice = append(HeightSlice, u.Height)
	}

	sort.Slice(HeightSlice, func(i, j int) bool {
		return HeightSlice[i] > HeightSlice[j]
	})
}

func LookupMethodInfo(epoch abi.ChainEpoch, Method abi.MethodNum, from, to string, Actor string) (actor.MethodInfo, error) {
	To, err := address.NewFromString(common.AddAddressPrefix(to))
	if err != nil {
		return actor.MethodInfo{}, err
	}
	if Method == lbuiltin.MethodSend {
		return actor.MethodSend, nil
	}

	lock.Lock()
	defer lock.Unlock()
	var actorSet *ActorSet
	for _, h := range HeightSlice {
		if epoch >= h {
			actorSet = UpgradeActorSet[h]
			break
		}
	}

	code := cid.Undef
	if code == cid.Undef {
		actorSet.loadmu.RLock()
		found, ok := actorSet.m[To]
		actorSet.loadmu.RUnlock()

		if ok {
			code = found
		}
	}

	var actType string
	if code == cid.Undef {
		actTypes := strings.Split(Actor, "/")
		actType = actTypes[2]
		av, err := strconv.Atoi(actTypes[1])
		if err != nil {
			return actor.MethodInfo{}, err
		}
		Code, err := GetBuiltinActorCodeID(actors.Version(av), actType)
		if err != nil {
			return actor.MethodInfo{}, fmt.Errorf("fallback to load from StateManager, still failed: %w", err)
		}
		actorSet.loadmu.Lock()
		actorSet.m[To] = Code
		actorSet.loadmu.Unlock()

		code = Code
	}

	vma := filcns.NewActorRegistry()
	mi, ok := vma.Methods[code][Method]
	if !ok {
		return actor.MethodInfo{}, fmt.Errorf("%w: lookup method for from=%s, to=%s, code=%s, meth=%d", actor.ErrActorMethodNotFound, from, To, code, Method)
	}

	return actor.MethodInfo{
		Actor:  actType,
		Method: mi,
	}, nil
}

func GetBuiltinActorCodeID(av actors.Version, actorName string) (cid.Cid, error) {
	//GetBuiltinActorsKeys
	switch actorName {
	case actors.AccountKey:
		code, err := lbuiltin.GetAccountActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.CronKey:
		code, err := lbuiltin.GetCronActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.InitKey:
		code, err := lbuiltin.GetInitActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.MarketKey:
		code, err := lbuiltin.GetMarketActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.MinerKey:
		code, err := lbuiltin.GetMinerActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.MultisigKey:
		code, err := lbuiltin.GetMultisigActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.PaychKey:
		code, err := lbuiltin.GetPaymentChannelActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.PowerKey:
		code, err := lbuiltin.GetPowerActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.RewardKey:
		code, err := lbuiltin.GetRewardActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.SystemKey:
		code, err := lbuiltin.GetSystemActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	case actors.VerifregKey:
		code, err := lbuiltin.GetVerifregActorCodeID(av)
		if err != nil {
			return cid.Undef, err
		}
		return code, nil
	default:
		return cid.Undef, fmt.Errorf("unknow actor name: %v", actorName)
	}
}
