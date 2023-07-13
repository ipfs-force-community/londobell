package segment

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment/model"
)

// State from segments
type State struct {
	dbState     *model.DBState // todo：指针？
	blockStates []model.SegmentState

	// only for colds now
	blockMethodStates         []model.SegmentState
	actorStates               []model.SegmentState
	actorMethodStates         []model.SegmentState
	actorTransferStates       []model.SegmentState
	minedStates               []model.SegmentState
	largeAmountTransferStates []model.SegmentState

	dealState model.DealState
	//dealActorStates []model.SegmentDealState
}

func DefaultState(dsn string, dtype model.DType, interval int64, startEpoch, endEpoch abi.ChainEpoch, startDealID, endDealID uint64) *State {
	dbState := model.NewDBState(dsn, dtype, interval, startEpoch, endEpoch)
	dealState := model.NewDealState(dsn, dtype, model.NoneInterval, startDealID, endDealID, 0)
	blockStates := make([]model.SegmentState, 0)
	return &State{
		dbState:     dbState,    // todo
		dealState:   *dealState, // todo
		blockStates: blockStates,
	}
}

func (s *State) GetDBState() *model.DBState {
	return s.dbState
}

func (s *State) GetDType() model.DType {
	return s.dbState.DType
}

func (s *State) GetDSN() string {
	return s.dbState.Dsn
}

func (s *State) GetStartEpoch() abi.ChainEpoch {
	return s.dbState.StartEpoch
}

func (s *State) GetEndEpoch() abi.ChainEpoch {
	return s.dbState.EndEpoch
}

func (s *State) GetDealState() model.DealState {
	return s.dealState
}

func (s *State) GetDealStartID() uint64 {
	return s.dealState.StartDealID
}

func (s *State) GetDealEndID() uint64 {
	return s.dealState.EndDealID
}

func (s *State) GetBlockStates() []model.SegmentState {
	return s.blockStates
}

func (s *State) GetBlockMethodStates(methodName string) []model.SegmentState {
	blockMethodStates := make([]model.SegmentState, 0)
	for _, bms := range s.blockMethodStates {
		if bms.MethodName == methodName {
			blockMethodStates = append(blockMethodStates, bms)
		}
	}

	return blockMethodStates
}

func (s *State) GetAllBlockMethodStates() []model.SegmentState {
	return s.blockMethodStates
}

func (s *State) GetActorStates(actorID string) []model.SegmentState {
	actorStates := make([]model.SegmentState, 0)
	for _, as := range s.actorStates {
		if as.ActorID == actorID {
			actorStates = append(actorStates, as)
		}
	}

	return actorStates
}

func (s *State) GetAllActorStates() []model.SegmentState {
	return s.actorStates
}

func (s *State) GetActorMethodStates(actorID string, methodName string) []model.SegmentState {
	actorMethodStates := make([]model.SegmentState, 0)
	for _, ams := range s.actorMethodStates {
		if ams.ActorID == actorID && ams.MethodName == methodName {
			actorMethodStates = append(actorMethodStates, ams)
		}
	}

	return actorMethodStates
}

func (s *State) GetAllActorMethodStates() []model.SegmentState {
	return s.GetAllBlockMethodStates()
}

func (s *State) GetActorTransferStates(actorID string) []model.SegmentState {
	actorTransferStates := make([]model.SegmentState, 0)
	for _, ats := range s.actorTransferStates {
		if ats.ActorID == actorID {
			actorTransferStates = append(actorTransferStates, ats)
		}
	}

	return actorTransferStates
}

func (s *State) GetAllActorTransferStates() []model.SegmentState {
	return s.actorTransferStates
}

func (s *State) GetMinedStates(actorID string) []model.SegmentState {
	minedStates := make([]model.SegmentState, 0)
	for _, ms := range s.minedStates {
		if ms.ActorID == actorID {
			minedStates = append(minedStates, ms)
		}
	}

	return minedStates
}

func (s *State) GetAllMinedStates() []model.SegmentState {
	return s.minedStates
}

func (s *State) GetLargeAmountTransferStates() []model.SegmentState {
	return s.largeAmountTransferStates
}

//func (s *State) GetDealActorStates(actorID string) []model.SegmentDealState {
//	dealActorStates := make([]model.SegmentDealState, 0)
//	for _, das := range s.dealActorStates {
//		if das.ActorID == actorID {
//			dealActorStates = append(dealActorStates, das)
//		}
//	}
//
//	return s.dealActorStates
//}
//
//func (s *State) GetAllDealActorStates() []model.SegmentDealState {
//	return s.dealActorStates
//}

func (s *State) SetEndEpoch(endEpoch abi.ChainEpoch) {
	s.dbState.EndEpoch = endEpoch
}

func (s *State) SetStartEpoch(startEpoch abi.ChainEpoch) {
	s.dbState.StartEpoch = startEpoch
}

func (s *State) SetDBState(dbState *model.DBState) {
	s.dbState = dbState
}

func (s *State) SetDealState(dealState model.DealState) {
	s.dealState = dealState
}

func (s *State) SetBlockStates(blockStates []model.SegmentState) error {
	//if blockStates == nil || len(blockStates) == 0 {
	//	return fmt.Errorf("null blockStates is not allowed to set")
	//}

	s.blockStates = blockStates
	return nil
}

func (s *State) SetBlockMethodStates(blockMethodStates []model.SegmentState) error {
	s.blockMethodStates = blockMethodStates
	return nil
}

func (s *State) SetActorStates(actorStates []model.SegmentState) error {
	s.actorStates = actorStates
	return nil
}

func (s *State) SetActorMethodStates(actorMethodStates []model.SegmentState) error {
	s.actorMethodStates = actorMethodStates
	return nil
}

func (s *State) SetActorTransferStates(actorTransferStates []model.SegmentState) error {
	s.actorTransferStates = actorTransferStates
	return nil
}

func (s *State) SetMinedStates(MinedStates []model.SegmentState) error {
	s.minedStates = MinedStates
	return nil
}

func (s *State) SetLargeAmountTransferStates(largeAmountTransferStates []model.SegmentState) error {
	s.largeAmountTransferStates = largeAmountTransferStates
	return nil
}

//func (s *State) SetDealActorStates(dealActorStates []model.SegmentDealState) error {
//	s.dealActorStates = dealActorStates
//	return nil
//}
