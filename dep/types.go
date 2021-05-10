package dep

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("dep")

// GlobalContext is a type alias for standard context.Context that used in dep-injection
type GlobalContext context.Context

// MgoMetaMgrDSN is the mongo dsn for metamgr
type MgoMetaMgrDSN string
