package reg

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/runtime"
	"github.com/ipfs-force-community/custom-actors-parsing/external"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"

	v8 "github.com/ipfs-force-community/custom-actors-parsing/external/v8"
	eam8 "github.com/ipfs-force-community/custom-actors-parsing/external/v8/eam"
	evm8 "github.com/ipfs-force-community/custom-actors-parsing/external/v8/evm"

	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/aerrors"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/vm"
)

var ActorReg = filcns.NewActorRegistry()

func NewExternalActorRegistry() *ActorRegistry {
	inv := NewActorRegistry()
	Register(inv, actors.Version8, ActorsVersionPredicate(actors.Version8), MakeRegistry(actors.Version8))

	return inv
}

func IsEam(c cid.Cid) bool {
	name, _, ok := actors.GetActorMetaByCode(c)
	if ok {
		return name == "eam"
	}

	return false
}

func DumpExternalActorState(i *ActorRegistry, act *types.Actor, b []byte) (interface{}, error) {
	if IsEam(act.Code) {
		return nil, nil
	}

	actorInfo, ok := i.actors[act.Code]
	if !ok {
		return nil, fmt.Errorf("state type for actor %s not found", act.Code)
	}

	um := actorInfo.vmActor.state
	if err := um.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("unmarshaling actor state: %w", err)
	}

	return um, nil
}

type invokeFunc func(rt runtime.Runtime, params []byte) ([]byte, aerrors.ActorError)
type nativeCode map[uint64]invokeFunc
type ActorPredicate func(runtime.Runtime, cid.Cid) error

type actorInfo struct {
	methods nativeCode
	vmActor RegistryEntry
	// TODO: consider making this a network version range?
	predicate ActorPredicate
}

type ActorRegistry struct {
	actors map[cid.Cid]*actorInfo

	Methods map[cid.Cid]map[abi.MethodNum]vm.MethodMeta
}

func ActorsVersionPredicate(ver actors.Version) ActorPredicate {
	return func(rt runtime.Runtime, codeCid cid.Cid) error {
		aver, err := actors.VersionForNetwork(rt.NetworkVersion())
		if err != nil {
			return fmt.Errorf("unsupported network version: %w", err)
		}
		if aver != ver {
			return fmt.Errorf("actor %s is a version %d actor; chain only supports actor version %d at height %d and nver %d", codeCid, ver, aver, rt.CurrEpoch(), rt.NetworkVersion())
		}
		return nil
	}
}

func NewActorRegistry() *ActorRegistry {
	return &ActorRegistry{
		actors:  make(map[cid.Cid]*actorInfo),
		Methods: map[cid.Cid]map[abi.MethodNum]vm.MethodMeta{},
	}
}

func Register(ar *ActorRegistry, av actors.Version, pred ActorPredicate, vmactors []RegistryEntry) {
	if pred == nil {
		pred = func(runtime.Runtime, cid.Cid) error { return nil }
	}
	for _, a := range vmactors {

		if av <= actors.Version7 {
			panic("expected version v8 and up only, use specs-actors for v0-7")
		}

		var code nativeCode
		ai := &actorInfo{
			methods:   code,
			vmActor:   a,
			predicate: pred,
		}
		ac := a.Code()
		// necessary to make stuff work
		var realCode cid.Cid
		if av >= actors.Version8 {
			name := actors.CanonicalName(builtin.ActorNameByCode(ac))

			var ok bool
			realCode, ok = actors.GetActorCodeID(av, name)
			if ok {
				ar.actors[realCode] = ai
			}
		}

		// register in the `Methods` map (used by statemanager utils)
		exports := a.Exports()
		methods := make(map[abi.MethodNum]vm.MethodMeta, len(exports))

		// Explicitly add send, it's special.
		methods[builtin.MethodSend] = vm.MethodMeta{
			Name:   "Send",
			Params: reflect.TypeOf(new(abi.EmptyValue)),
			Ret:    reflect.TypeOf(new(abi.EmptyValue)),
		}

		// Iterate over exported methods. Some of these _may_ be nil and
		// must be skipped.
		for number, export := range exports {
			if export.Method == nil {
				continue
			}

			ev := reflect.ValueOf(export.Method)
			et := ev.Type()

			mm := vm.MethodMeta{
				Name: export.Name,
				Ret:  et.Out(0),
			}

			// methods exported from go-state-types do not, so we want et.In(0)
			mm.Params = et.In(0)

			methods[abi.MethodNum(number)] = mm
		}
		if realCode.Defined() {
			ar.Methods[realCode] = methods
		} else {
			ar.Methods[a.Code()] = methods
		}
	}
}

type RegistryEntry struct {
	state   cbor.Er
	code    cid.Cid
	methods map[uint64]external.MethodMeta
}

func (r RegistryEntry) State() cbor.Er {
	return r.state
}

func (r RegistryEntry) Exports() map[uint64]external.MethodMeta {
	return r.methods
}

func (r RegistryEntry) Code() cid.Cid {
	return r.code
}

func MakeRegistry(av actors.Version) []RegistryEntry {
	if av < actors.Version8 {
		panic("expected version v8 and up only, use specs-actors for v0-7")
	}

	registry := make([]RegistryEntry, 0)

	codeIDs, err := actors.GetActorCodeIDs(av)
	if err != nil {
		panic(err)
	}

	switch av {
	case actors.Version8:
		for key, codeID := range codeIDs {
			switch key {
			case v8.EamKey:
				registry = append(registry, RegistryEntry{
					code:    codeID,
					methods: eam8.Methods,
					state:   nil,
				})
			case v8.EvmKey:
				registry = append(registry, RegistryEntry{
					code:    codeID,
					methods: evm8.Methods,
					state:   new(evm8.State),
				})
			case v8.EmbryoKey:
				registry = append(registry, RegistryEntry{
					code:    codeID,
					methods: nil,
					state:   nil,
				})
			}
		}
	default:
		panic("expected version v8 and up only, use specs-actors for v0-7")
	}

	return registry
}
