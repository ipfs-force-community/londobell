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
	segmentActiveKey       = datastore.NewKey("active")
	namespaceSeg           = datastore.NewKey("/manager")
	namespaceDataBaseInfo  = datastore.NewKey("/info")
	namespaceDataBaseState = datastore.NewKey("/state")
)

func NewStateManager(mds dtypes.MetadataDS) (*StateManager, error) {
	base := namespace.Wrap(mds, namespaceSeg)
	info := namespace.Wrap(base, namespaceDataBaseInfo)
	state := namespace.Wrap(base, namespaceDataBaseState)
	return &StateManager{
		base:  base,
		info:  info,
		state: state,
	}, nil
}

type Info struct {
	DSN struct {
		NewWrite []string
	}
}

type StateManager struct {
	base  datastore.Batching
	info  datastore.Batching // DataBase
	state datastore.Batching //DataBaseState
}

func (m *StateManager) SetActive(name string) error {
	return m.base.Put(context.Background(), segmentActiveKey, []byte(name))
}

func (m *StateManager) LoadActive() (string, bool, error) {
	data, err := m.base.Get(context.Background(), segmentActiveKey)
	if err == datastore.ErrNotFound {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return string(data), true, nil
}

func (m *StateManager) LoadInfo(name string) (Info, bool, error) {
	data, err := m.info.Get(context.Background(), datastore.NewKey(name))
	if err == datastore.ErrNotFound {
		return Info{}, false, nil
	}

	if err != nil {
		return Info{}, false, err
	}

	var info Info
	err = json.Unmarshal(data, &info)
	return info, true, err
}

func (m *StateManager) SetInfo(name string, info Info) error {
	data, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}

	return m.info.Put(context.Background(), datastore.NewKey(name), data)
}

//// todo: 线程安全？
//func (m *StateManager) SetDataBaseState(url string, b DataBaseState) error {
//	data, err := json.Marshal(&b)
//	if err != nil {
//		return fmt.Errorf("marshal boundary: %w", err)
//	}
//
//	return m.state.Put(context.Background(), datastore.NewKey(url), data)
//}
//
//func (m *StateManager) LoadDataBaseState(url string) (DataBaseState, bool, error) {
//	data, err := m.state.Get(context.Background(), datastore.NewKey(url))
//	if err == datastore.ErrNotFound {
//		return DataBaseState{}, false, nil
//	}
//
//	if err != nil {
//		return DataBaseState{}, false, err
//	}
//
//	var b DataBaseState
//	err = json.Unmarshal(data, &b)
//	return b, true, err
//}
//
//
//func (m *StateManager) DeleteDataBaseState(url string) error {
//	return m.state.Delete(context.Background(), datastore.NewKey(url))
//}
