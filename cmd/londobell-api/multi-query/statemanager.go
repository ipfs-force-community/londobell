package multiquery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/ipfs/go-datastore/namespace"

	"github.com/ipfs/go-datastore"
)

var (
	namespaceSeg = datastore.NewKey("/manager")
	//namespaceDataBaseInfo  = datastore.NewKey("/info")
	namespaceDataBaseState = datastore.NewKey("/state")
)

func NewStateManager(mds dtypes.MetadataDS) (*StateManager, error) {
	base := namespace.Wrap(mds, namespaceSeg)
	//info := namespace.Wrap(base, namespaceDataBaseInfo)
	state := namespace.Wrap(base, namespaceDataBaseState)
	return &StateManager{
		base: base,
		//info:  info,
		state: state,
	}, nil
}

type StateManager struct {
	base datastore.Batching
	//info  datastore.Batching // DataBase
	state datastore.Batching //DataBaseState
}

// todo: 线程安全？
func (m *StateManager) SetDataBaseState(url string, b DataBaseState) error {
	data, err := json.Marshal(&b)
	if err != nil {
		return fmt.Errorf("marshal boundary: %w", err)
	}

	return m.state.Put(context.Background(), datastore.NewKey(url), data)
}

func (m *StateManager) LoadDataBaseState(url string) (DataBaseState, bool, error) {
	data, err := m.state.Get(context.Background(), datastore.NewKey(url))
	if err == datastore.ErrNotFound {
		return DataBaseState{}, false, nil
	}

	if err != nil {
		return DataBaseState{}, false, err
	}

	var b DataBaseState
	err = json.Unmarshal(data, &b)
	return b, true, err
}
