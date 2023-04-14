package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lotus/chain/actors"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
	"github.com/ipfs/go-cid"
)

var Nodes []Node

type Node struct {
	URL   string `json:"node"`
	Token string `json:"token"`
}

func ParseNodes(path string) error {
	file, err := os.Open(path)
	defer file.Close() //nolint:staticcheck
	if err != nil {
		panic(err)
	}

	configByte, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configByte, &Nodes)
	if err != nil {
		panic(err)
	}

	return nil
}

type ActorSet struct {
	m      map[address.Address]cid.Cid
	loadmu sync.RWMutex
}

func NewActorSet() *ActorSet {
	m := make(map[address.Address]cid.Cid)
	return &ActorSet{m: m}
}

func LookupMethodInfo(epoch abi.ChainEpoch, Method abi.MethodNum, from, to string, Actor string) (actor.MethodInfo, error) {
	To, err := address.NewFromString(common.AddAddressPrefix(to))
	if err != nil {
		return actor.MethodInfo{}, err
	}
	if Method == lbuiltin.MethodSend {
		return actor.MethodSend, nil
	}

	actorSet := NewActorSet()

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
		if Actor == "" {
			return actor.MethodInfo{}, nil
		}
		actTypes := strings.Split(Actor, "/")
		if len(actTypes) != 3 {
			return actor.MethodInfo{}, fmt.Errorf("length of acttypes is not equal 3, epoch: %v, Method:%v, from: %v, to: %v, Actor: %v", epoch, Method, from, to, Actor)
		}

		actType = actTypes[2]
		av, err := strconv.Atoi(actTypes[1])
		if err != nil {
			return actor.MethodInfo{}, err
		}
		actorCode, err := GetBuiltinActorCodeID(actorstypes.Version(av), actType)
		if err != nil {
			return actor.MethodInfo{}, fmt.Errorf("fallback to load from StateManager, still failed: %w", err)
		}
		actorSet.loadmu.Lock()
		actorSet.m[To] = actorCode
		actorSet.loadmu.Unlock()

		code = actorCode
	}

	if ccode, cname, ok := actor.DefaultActorConvertor(epoch, Actor); ok {
		code = ccode
		Actor = cname
	}

	vma := filcns.NewActorRegistry()
	mi, ok := vma.Methods[code][Method]
	if !ok {
		return actor.MethodInfo{}, fmt.Errorf("%w: lookup method for from=%s, to=%s, code=%s, meth=%d", actor.ErrActorMethodNotFound, from, To, code, Method)
	}

	return actor.MethodInfo{
		Actor:  Actor,
		Method: mi,
	}, nil
}

func GetBuiltinActorCodeID(av actorstypes.Version, actorName string) (cid.Cid, error) {
	// GetBuiltinActorsKeys
	codeIDs, err := actors.GetActorCodeIDs(av)
	if err != nil {
		return cid.Undef, err
	}

	if _, ok := codeIDs[actorName]; !ok {
		return cid.Undef, fmt.Errorf("unknow actor type: %v", actorName)
	}

	return codeIDs[actorName], nil
}

type StopFuncMap struct {
	sync.RWMutex
	stop map[string]dix.StopFunc
}

var stopFuncMap = &StopFuncMap{stop: make(map[string]dix.StopFunc)}

func RegistryStopFuncMap(url string, stopFunc dix.StopFunc) {
	stopFuncMap.Lock()
	stopFuncMap.stop[url] = stopFunc
	stopFuncMap.Unlock()
}

func GetStopFuncByUrl(url string) dix.StopFunc {
	stopFuncMap.RLock()
	defer stopFuncMap.RUnlock()
	return stopFuncMap.stop[url]
}
