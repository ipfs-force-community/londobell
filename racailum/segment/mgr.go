package segment

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"

	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	segmentActiveKey = datastore.NewKey("active")

	namespaceSeg      = datastore.NewKey("/seg")
	namespaceSegInfo  = datastore.NewKey("/info")
	namespaceSegBound = datastore.NewKey("/bound")
)

// Boundary marks the high & low bound of the segment
type Boundary struct {
	Hi Anchor
	Lo Anchor
}

// SetHi set the high bound to the given tipset
func (b *Boundary) SetHi(ts *common.LinkedTipSet) {
	b.Hi = Anchor{
		Epoch: ts.Height(),
		TSK:   ts.Key(),
		State: ts.State(),
	}
}

// SetLo set the high bound to the given tipset
func (b *Boundary) SetLo(ts *common.LinkedTipSet) {
	b.Lo = Anchor{
		Epoch: ts.Height(),
		TSK:   ts.Key(),
		State: ts.State(),
	}
}

func NewManager(mds dtypes.MetadataDS) (*Manager, error) {
	base := namespace.Wrap(mds, namespaceSeg)
	info := namespace.Wrap(base, namespaceSegInfo)
	bound := namespace.Wrap(base, namespaceSegBound)
	return &Manager{
		base:  base,
		info:  info,
		bound: bound,
	}, nil
}

type Info struct {
	DSN struct {
		Write    string
		Read     string
		NewWrite []string
	}
}

type Manager struct {
	base  datastore.Batching
	info  datastore.Batching
	bound datastore.Batching
}

func (m *Manager) SetActive(name string) error {
	return m.base.Put(segmentActiveKey, []byte(name))
}

func (m *Manager) LoadActive() (string, bool, error) {
	data, err := m.base.Get(segmentActiveKey)
	if err == datastore.ErrNotFound {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return string(data), true, nil
}

func (m *Manager) SetBoundary(name string, b Boundary) error {
	data, err := json.Marshal(&b)
	if err != nil {
		return fmt.Errorf("marshal boundary: %w", err)
	}

	return m.bound.Put(datastore.NewKey(name), data)
}

func (m *Manager) LoadBoundary(name string) (Boundary, bool, error) {
	data, err := m.bound.Get(datastore.NewKey(name))
	if err == datastore.ErrNotFound {
		return Boundary{}, false, nil
	}

	if err != nil {
		return Boundary{}, false, err
	}

	var b Boundary
	err = json.Unmarshal(data, &b)
	return b, true, err
}

func (m *Manager) SetInfo(name string, info Info) error {
	data, err := json.Marshal(&info)
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}

	return m.info.Put(datastore.NewKey(name), data)
}

func (m *Manager) LoadInfo(name string) (Info, bool, error) {
	data, err := m.info.Get(datastore.NewKey(name))
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
