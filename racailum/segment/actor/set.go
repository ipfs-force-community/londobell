package actor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dtynn/londobell/common"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	ainit "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	linit "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("actor")

// common errors
var (
	ErrActorMethodNotFound = fmt.Errorf("actor method not found")
)

// NewSet loads actor codes and construct a actor set with the given tipset
func NewSet(ctx context.Context, stm common.StateManager, ts *common.LinkedTipSet) (*Set, error) {
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
		if aid, err := address.IDFromAddress(call.To); err == nil && aid > builtin.FirstNonSingletonActorId {
			switch {
			case parent.To == builtin.InitActorAddr && parent.Method == builtin.MethodsInit.Exec:
				parentParam := &ainit.ExecParams{}
				if err := parentParam.UnmarshalCBOR(bytes.NewReader(parent.Params)); err == nil {
					code = parentParam.CodeCID
				}

			case call.From == lbuiltin.SystemActorAddr && parent.Method == lbuiltin.MethodSend:
				code = builtin.AccountActorCodeID

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

	return MethodInfo{
		Actor:  actorName,
		Method: mi,
	}, nil
}
