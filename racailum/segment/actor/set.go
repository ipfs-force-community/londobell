package actor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lotus/chain/actors"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	"github.com/filecoin-project/specs-actors/v8/actors/builtin"
	"go.opencensus.io/trace"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	ainit "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	linit "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/common"
)

var log = logging.Logger("actor")

// common errors
var (
	ErrActorMethodNotFound = fmt.Errorf("actor method not found")
)

// NewSet loads actor codes and construct a actor set with the given tipset
func NewSet(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet) (*Set, error) {
	_, span := trace.StartSpan(ctx, "actor.NewSet")
	defer span.End()

	m := map[address.Address]cid.Cid{}

	root := ts.State()
	tree, err := stm.StateTree(root)
	if err != nil {
		return nil, fmt.Errorf("load state tree: %w", err)
	}

	// load actor idss
	var initActor *types.Actor

	count := 0
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		m[addr] = act.Code
		count++

		if addr == builtin.InitActorAddr {
			initActor = act
		}

		return nil
	}); err != nil {
		return nil, err
	}

	initCount := 0
	if initActor != nil {
		ias, err := linit.Load(&state.AdtStore{IpldStore: tree.Store}, initActor)
		if err != nil {
			return nil, fmt.Errorf("load init state: %w", err)
		}

		if err := ias.ForEachActor(func(id abi.ActorID, addr address.Address) error {
			initCount++

			idAddr, err := address.NewIDAddress(uint64(id))
			if err != nil {
				return fmt.Errorf("generate id addr for %d: %w", id, err)
			}

			code, ok := m[idAddr]
			if ok {
				m[addr] = code
			} else {
				log.Warnf("code not found for actor id %d, but exists in init state", id)
			}

			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk through actor init state: %w", err)
		}
	}

	log.Infow("actor set loaded", "epoch", ts.Height(), "state", root, "count", count, "init-state", initCount, "total", len(m))

	return &Set{m: m}, nil
}

// Set loads actor codes from a given tipset
type Set struct {
	m      map[address.Address]cid.Cid
	loadmu sync.RWMutex
}

// LookupMethodInfo returns method info for the given message along with its parent if any
func (s *Set) LookupMethodInfo(ctx context.Context, ts *types.TipSet, stm common.StateManager, parent, call *types.Message) (MethodInfo, error) {
	if call.Method == lbuiltin.MethodSend {
		return MethodSend, nil
	}

	code := cid.Undef

	// for MethodConstructor subcalls, we should look into it's parent call
	// as aborted execution of its' parent message would rollback the actor id assignment, the final actor code may be mismatched
	if call.Method == lbuiltin.MethodConstructor && call.To.Protocol() == address.ID && parent != nil {
		if aid, err := address.IDFromAddress(call.To); err == nil && aid > builtin3.FirstNonSingletonActorId {
			switch {
			case parent.To == builtin3.InitActorAddr && parent.Method == builtin3.MethodsInit.Exec:
				parentParam := &ainit.ExecParams{}
				if err := parentParam.UnmarshalCBOR(bytes.NewReader(parent.Params)); err == nil {
					code = parentParam.CodeCID
				}

			case call.From == lbuiltin.SystemActorAddr && parent.Method == lbuiltin.MethodSend:
				code = builtin3.AccountActorCodeID

			}
		}
	}

	// get from pre loaded actor set
	if code == cid.Undef {
		s.loadmu.RLock()
		found, ok := s.m[call.To]
		s.loadmu.RUnlock()

		if ok {
			code = found
		}
	}

	// fall back to state inside tipset
	if code == cid.Undef {
		act, err := stm.LoadActor(ctx, call.To, ts)
		if err != nil {
			if errors.Is(err, types.ErrActorNotFound) {
				return MethodInfo{}, fmt.Errorf("%w: %s", ErrActorMethodNotFound, err)
			}

			return MethodInfo{}, fmt.Errorf("fallback to load from StateManager, still failed: %w", err)
		}

		s.loadmu.Lock()
		log.Warnf("fallback to load actor code for %s, got %s", call.To, act.Code)
		s.m[call.To] = act.Code
		s.loadmu.Unlock()

		code = act.Code
	}

	actorName := lbuiltin.ActorNameByCode(code)

	if ccode, cname, ok := DefaultActorConvertor(ts.Height(), actorName); ok {
		code = ccode
		actorName = cname
	}
	vma := filcns.NewActorRegistry()

	mi, ok := vma.Methods[code][call.Method]
	if !ok {
		return MethodInfo{}, fmt.Errorf("%w: lookup method for from=%s, to=%s, code=%s, meth=%d", ErrActorMethodNotFound, call.From, call.To, code, call.Method)
	}

	methodName := mi.Num
	methods, err := loadActorMethods(code)
	if err != nil {
		log.Warnf("load actor methods failed: %v", err)
		return MethodInfo{
			Actor:      actorName,
			Method:     mi,
			MethodName: methodName,
		}, nil
	}

	mv := reflect.ValueOf(methods)
	mt := mv.Type()
	num, err := strconv.Atoi(mi.Num)
	if err != nil {
		log.Warnf("converse Num %v for (to %v, code %v) failed: %v", mi.Num, call.To, code, err)
		return MethodInfo{
			Actor:      actorName,
			Method:     mi,
			MethodName: methodName,
		}, nil
	}

	if num <= 0 || num >= mt.NumField() {
		log.Warnf("Num %v is not included in [0, %v] for (to %v, code %v) except MethodSend", num, mt.NumField()-1, call.To, code)
		return MethodInfo{
			Actor:      actorName,
			Method:     mi,
			MethodName: methodName,
		}, nil
	}
	methodName = mt.Field(num - 1).Name

	return MethodInfo{
		Actor:      actorName,
		Method:     mi,
		MethodName: methodName,
	}, nil
}

func loadActorMethods(code cid.Cid) (interface{}, error) {
	if name, av, ok := actors.GetActorMetaByCode(code); ok {
		switch av {
		case actorstypes.Version8, actorstypes.Version9:
			switch name {
			case actors.AccountKey:
				return sbuiltin.MethodsAccount, nil
			case actors.CronKey:
				return sbuiltin.MethodsCron, nil
			case actors.InitKey:
				return sbuiltin.MethodsInit, nil
			case actors.MarketKey:
				return sbuiltin.MethodsMarket, nil
			case actors.MinerKey:
				return sbuiltin.MethodsMiner, nil
			case actors.MultisigKey:
				return sbuiltin.MethodsMultisig, nil
			case actors.PaychKey:
				return sbuiltin.MethodsPaych, nil
			case actors.PowerKey:
				return sbuiltin.MethodsPower, nil
			case actors.RewardKey:
				return sbuiltin.MethodsReward, nil
			case actors.SystemKey:
				return struct{ Constructor abi.MethodNum }{sbuiltin.MethodConstructor}, nil
			case actors.VerifregKey:
				return sbuiltin.MethodsVerifiedRegistry, nil
			default:
				return nil, fmt.Errorf("unknow butiltin actor type")
			}
		}
	}
	switch code {
	//v0
	case builtin0.AccountActorCodeID:
		return builtin0.MethodsAccount, nil
	case builtin0.CronActorCodeID:
		return builtin0.MethodsCron, nil
	case builtin0.InitActorCodeID:
		return builtin0.MethodsInit, nil
	case builtin0.StorageMarketActorCodeID:
		return builtin0.MethodsMarket, nil
	case builtin0.StorageMinerActorCodeID:
		return builtin0.MethodsMiner, nil
	case builtin0.MultisigActorCodeID:
		return builtin0.MethodsMultisig, nil
	case builtin0.PaymentChannelActorCodeID:
		return builtin0.MethodsPaych, nil
	case builtin0.StoragePowerActorCodeID:
		return builtin0.MethodsPower, nil
	case builtin0.RewardActorCodeID:
		return builtin0.MethodsReward, nil
	case builtin0.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin0.MethodConstructor}, nil
	case builtin0.VerifiedRegistryActorCodeID:
		return builtin0.MethodsVerifiedRegistry, nil
	//v2
	case builtin2.AccountActorCodeID:
		return builtin2.MethodsAccount, nil
	case builtin2.CronActorCodeID:
		return builtin2.MethodsCron, nil
	case builtin2.InitActorCodeID:
		return builtin2.MethodsInit, nil
	case builtin2.StorageMarketActorCodeID:
		return builtin2.MethodsMarket, nil
	case builtin2.StorageMinerActorCodeID:
		return builtin2.MethodsMiner, nil
	case builtin2.MultisigActorCodeID:
		return builtin2.MethodsMultisig, nil
	case builtin2.PaymentChannelActorCodeID:
		return builtin2.MethodsPaych, nil
	case builtin2.StoragePowerActorCodeID:
		return builtin2.MethodsPower, nil
	case builtin2.RewardActorCodeID:
		return builtin2.MethodsReward, nil
	case builtin2.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin2.MethodConstructor}, nil
	case builtin2.VerifiedRegistryActorCodeID:
		return builtin2.MethodsVerifiedRegistry, nil
	//v3
	case builtin3.AccountActorCodeID:
		return builtin3.MethodsAccount, nil
	case builtin3.CronActorCodeID:
		return builtin3.MethodsCron, nil
	case builtin3.InitActorCodeID:
		return builtin3.MethodsInit, nil
	case builtin3.StorageMarketActorCodeID:
		return builtin3.MethodsMarket, nil
	case builtin3.StorageMinerActorCodeID:
		return builtin3.MethodsMiner, nil
	case builtin3.MultisigActorCodeID:
		return builtin3.MethodsMultisig, nil
	case builtin3.PaymentChannelActorCodeID:
		return builtin3.MethodsPaych, nil
	case builtin3.StoragePowerActorCodeID:
		return builtin3.MethodsPower, nil
	case builtin3.RewardActorCodeID:
		return builtin3.MethodsReward, nil
	case builtin3.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin3.MethodConstructor}, nil
	case builtin3.VerifiedRegistryActorCodeID:
		return builtin3.MethodsVerifiedRegistry, nil
	//v4
	case builtin4.AccountActorCodeID:
		return builtin4.MethodsAccount, nil
	case builtin4.CronActorCodeID:
		return builtin4.MethodsCron, nil
	case builtin4.InitActorCodeID:
		return builtin4.MethodsInit, nil
	case builtin4.StorageMarketActorCodeID:
		return builtin4.MethodsMarket, nil
	case builtin4.StorageMinerActorCodeID:
		return builtin4.MethodsMiner, nil
	case builtin4.MultisigActorCodeID:
		return builtin4.MethodsMultisig, nil
	case builtin4.PaymentChannelActorCodeID:
		return builtin4.MethodsPaych, nil
	case builtin4.StoragePowerActorCodeID:
		return builtin4.MethodsPower, nil
	case builtin4.RewardActorCodeID:
		return builtin4.MethodsReward, nil
	case builtin4.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin4.MethodConstructor}, nil
	case builtin4.VerifiedRegistryActorCodeID:
		return builtin4.MethodsVerifiedRegistry, nil
	//v5
	case builtin5.AccountActorCodeID:
		return builtin5.MethodsAccount, nil
	case builtin5.CronActorCodeID:
		return builtin5.MethodsCron, nil
	case builtin5.InitActorCodeID:
		return builtin5.MethodsInit, nil
	case builtin5.StorageMarketActorCodeID:
		return builtin5.MethodsMarket, nil
	case builtin5.StorageMinerActorCodeID:
		return builtin5.MethodsMiner, nil
	case builtin5.MultisigActorCodeID:
		return builtin5.MethodsMultisig, nil
	case builtin5.PaymentChannelActorCodeID:
		return builtin5.MethodsPaych, nil
	case builtin5.StoragePowerActorCodeID:
		return builtin5.MethodsPower, nil
	case builtin5.RewardActorCodeID:
		return builtin5.MethodsReward, nil
	case builtin5.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin5.MethodConstructor}, nil
	case builtin5.VerifiedRegistryActorCodeID:
		return builtin5.MethodsVerifiedRegistry, nil
	//v6
	case builtin6.AccountActorCodeID:
		return builtin6.MethodsAccount, nil
	case builtin6.CronActorCodeID:
		return builtin6.MethodsCron, nil
	case builtin6.InitActorCodeID:
		return builtin6.MethodsInit, nil
	case builtin6.StorageMarketActorCodeID:
		return builtin6.MethodsMarket, nil
	case builtin6.StorageMinerActorCodeID:
		return builtin6.MethodsMiner, nil
	case builtin6.MultisigActorCodeID:
		return builtin6.MethodsMultisig, nil
	case builtin6.PaymentChannelActorCodeID:
		return builtin6.MethodsPaych, nil
	case builtin6.StoragePowerActorCodeID:
		return builtin6.MethodsPower, nil
	case builtin6.RewardActorCodeID:
		return builtin6.MethodsReward, nil
	case builtin6.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin6.MethodConstructor}, nil
	case builtin6.VerifiedRegistryActorCodeID:
		return builtin6.MethodsVerifiedRegistry, nil
	//v7
	case builtin7.AccountActorCodeID:
		return builtin7.MethodsAccount, nil
	case builtin7.CronActorCodeID:
		return builtin6.MethodsCron, nil
	case builtin7.InitActorCodeID:
		return builtin7.MethodsInit, nil
	case builtin7.StorageMarketActorCodeID:
		return builtin7.MethodsMarket, nil
	case builtin7.StorageMinerActorCodeID:
		return builtin7.MethodsMiner, nil
	case builtin7.MultisigActorCodeID:
		return builtin7.MethodsMultisig, nil
	case builtin7.PaymentChannelActorCodeID:
		return builtin7.MethodsPaych, nil
	case builtin7.StoragePowerActorCodeID:
		return builtin7.MethodsPower, nil
	case builtin7.RewardActorCodeID:
		return builtin7.MethodsReward, nil
	case builtin7.SystemActorCodeID:
		return struct{ Constructor abi.MethodNum }{builtin7.MethodConstructor}, nil
	case builtin7.VerifiedRegistryActorCodeID:
		return builtin7.MethodsVerifiedRegistry, nil
	default:
		return nil, fmt.Errorf("unkonw builtin actor type")
	}
}
