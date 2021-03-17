package segment

import (
	"sync"

	"github.com/dtynn/londobell/common"
)

func NewDataSet(name string, db common.DocumentDB, metamgr common.MetaManager) (*DataSet, error) {
	panic("not impl")
}

// DataSet is a read-only set of collections of chain data, ranged by epoches
type DataSet struct {
	name    string
	boundMu sync.RWMutex
	bound   Boundary
	db      common.DocumentDB
}

// Bound returns the boundary of current data set
func (d *DataSet) Bound() Boundary {
	d.boundMu.RLock()
	b := d.bound
	d.boundMu.RUnlock()
	return b
}

// DB returns the read only DB instance of current data set
func (d *DataSet) DB() common.DocumentDB {
	return d.db
}
