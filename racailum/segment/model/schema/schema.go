package schema

import (
	"sync"

	"github.com/ipfs-force-community/londobell/common"
)

// Model is a named document
type Model struct {
	Name string
	D    common.Document
}

var models = struct {
	sync.RWMutex
	m []Model
}{}

// Models returns all registered models
func Models() []Model {
	models.RLock()
	m := models.m
	models.RUnlock()
	return m
}

// Register adds a document into the models
func Register(m ...Model) {
	models.Lock()
	models.m = append(models.m, m...)
	models.Unlock()
}
