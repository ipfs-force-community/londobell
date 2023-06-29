package segment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/ipfs/go-datastore/namespace"

	"github.com/ipfs/go-datastore"
)

var (
	segmentActiveKey = datastore.NewKey("active")

	namespaceSeg          = datastore.NewKey("/seg")
	namespaceDataBaseInfo = datastore.NewKey("/info")
)

func NewStateManager(mds dtypes.MetadataDS) (*SegManager, error) {
	base := namespace.Wrap(mds, namespaceSeg)
	info := namespace.Wrap(base, namespaceDataBaseInfo)
	return &SegManager{
		base: base,
		info: info,
	}, nil
}

type SegManager struct {
	base datastore.Batching
	info datastore.Batching
}

type Info struct {
	Write string //todo: 多写
	Read  string // formal & colds
}

func (m *SegManager) SetActive(name string) error {
	return m.base.Put(context.Background(), segmentActiveKey, []byte(name))
}

func (m *SegManager) LoadActive() (string, bool, error) {
	data, err := m.base.Get(context.Background(), segmentActiveKey)
	if err == datastore.ErrNotFound {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return string(data), true, nil
}

func (m *SegManager) SetInfo(name string, info Info) error {
	data, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}

	return m.info.Put(context.Background(), datastore.NewKey(name), data)
}

func (m *SegManager) LoadInfo(name string) (Info, bool, error) {
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
