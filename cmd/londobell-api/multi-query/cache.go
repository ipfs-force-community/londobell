package multiquery

import (
	"fmt"
	"sync"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
	smodel "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

type DataBaseStateCache struct {
	states map[string]*segment.State
	clk    sync.RWMutex
}

func NewDataBaseStateCache() *DataBaseStateCache {
	return &DataBaseStateCache{
		states: make(map[string]*segment.State),
	}
}

func (dbsc *DataBaseStateCache) GetState(dsn string) (*segment.State, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state, true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetDBState(dsn string) (*model.DBState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetDBState(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetBlockStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetBlockStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetBlockMethodStates(dsn string, methodName string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetBlockMethodStates(methodName), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetAllBlockMethodStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllBlockMethodStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetActorStates(dsn string, actorID string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetActorStates(actorID), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetAllActorStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllActorStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetActorMethodStates(dsn string, actorID string, methodName string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetActorMethodStates(actorID, methodName), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetAllActorMethodStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllActorMethodStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetActorTransferStates(dsn string, actorID string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetActorTransferStates(actorID), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetAllActorTransferStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllActorTransferStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetMinedStates(dsn string, actorID string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetMinedStates(actorID), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetAllMinedStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllMinedStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetLargeAmountTransferStates(dsn string) ([]model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetLargeAmountTransferStates(), true
	}

	return nil, false
}

func (dbsc *DataBaseStateCache) GetDealState(dsn string) (model.DealState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetDealState(), true
	}

	return model.DealState{}, false
}

func (dbsc *DataBaseStateCache) GetAllMethodNameState(dsn string) (model.SegmentState, bool) {
	dbsc.clk.RLock()
	defer dbsc.clk.RUnlock()

	if state, ok := dbsc.states[dsn]; ok {
		return state.GetAllMethodNameStates(), true
	}

	return model.SegmentState{}, false
}

//func (dbsc *DataBaseStateCache) GetAllDealActorStates(dsn string) (model.SegmentDealState, bool) {
//	dbsc.clk.RLock()
//	defer dbsc.clk.RUnlock()
//
//	if state, ok := dbsc.states[dsn]; ok {
//		return state.GetAllDealActorStates(), true
//	}
//
//	return nil, false
//}

//func (dbsc *DataBaseStateCache) GetDealActorStates(dsn string, actorID string) ([]model.SegmentDealState, bool) {
//	dbsc.clk.RLock()
//	defer dbsc.clk.RUnlock()
//
//	if state, ok := dbsc.states[dsn]; ok {
//		return state.GetDealActorStates(actorID), true
//	}
//
//	return nil, false
//}

func (dbsc *DataBaseStateCache) SetState(url string, state *segment.State) {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	dbsc.states[url] = state
}

func (dbsc *DataBaseStateCache) FindAndUpdateDBState(dsn string, dbState *smodel.DBState) {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		state.SetDBState(dbState)
		return
	}

	// todo: blockStates is nil
	state := &segment.State{}
	state.SetDBState(dbState)
	dbsc.SetState(dsn, state)
	return
}

func (dbsc *DataBaseStateCache) FindAndUpdateDealState(dsn string, dealState model.DealState) {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		state.SetDealState(dealState)
		return
	}

	state := &segment.State{}
	state.SetDealState(dealState)
	dbsc.SetState(dsn, state)
	return
}

func (dbsc *DataBaseStateCache) SetBlockStates(dsn string, blockStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetBlockStates(blockStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetBlockMethodStates(dsn string, blockMethodStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetBlockMethodStates(blockMethodStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetActorStates(dsn string, actorStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetActorStates(actorStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetActorMethodStates(dsn string, actorMethodStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetActorMethodStates(actorMethodStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetActorTransferStates(dsn string, actorTransferState []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetActorTransferStates(actorTransferState); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetMinedStates(dsn string, minedStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetMinedStates(minedStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetLargeAmountTransferStates(dsn string, largeAmountTransferStates []smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetLargeAmountTransferStates(largeAmountTransferStates); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

func (dbsc *DataBaseStateCache) SetAllMethodNameState(dsn string, allMethodNameState smodel.SegmentState) error {
	dbsc.clk.Lock()
	defer dbsc.clk.Unlock()

	if state, ok := dbsc.states[dsn]; ok {
		if err := state.SetAllMethodNameState(allMethodNameState); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("state of dsn %v not found", dsn)
}

//func (dbsc *DataBaseStateCache) SetDealActorStates(dsn string, dealActorStates []smodel.SegmentDealState) error {
//	dbsc.clk.Lock()
//	defer dbsc.clk.Unlock()
//
//	if state, ok := dbsc.states[dsn]; ok {
//		if err := state.SetDealActorStates(dealActorStates); err != nil {
//			return err
//		}
//
//		return nil
//	}
//
//	return fmt.Errorf("state of dsn %v not found", dsn)
//}
