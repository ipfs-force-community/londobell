package dep

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

var log = logging.Logger("dep")

// GlobalContext is a type alias for standard context.Context that used in dep-injection
type GlobalContext context.Context

// MgoMetaDSClient is an instance of connected mongo client used for metads and other components
type MgoMetaDSClient *mongo.Client

// MgoMetaDSDSN is the mongo dsn for metadata ds
type MgoMetaDSDSN string

// MgoMetaDSReadOnly defines if we use the metads in read only mode
type MgoMetaDSReadOnly bool

// MgoBstoreDSN is the mongo dsn for chain blockstore
type MgoBstoreDSN string

// MgoBstoreSync defines if we insert new data blocks in sync or async mode
type MgoBstoreSync bool

// MgoBstoreReadOnly defines if we use the blockstore in read only mode
type MgoBstoreReadOnly bool

// MgoMetaMgrDSN is the mongo dsn for metamgr
type MgoMetaMgrDSN string
