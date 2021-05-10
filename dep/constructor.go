package dep

import (
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/api"

	"github.com/dtynn/londobell/common"
	"github.com/dtynn/londobell/lib/cliex"
	"github.com/dtynn/londobell/lib/mgoutil"
	"github.com/dtynn/londobell/lib/mgoutil/mmetamgr"
	"github.com/dtynn/londobell/racailum"
	"github.com/dtynn/londobell/racailum/segment"
)

var (
	_ common.MetaManager  = (*mmetamgr.MetaMgr)(nil)
	_ common.DocumentDB   = (*mgoutil.MgoDocDB)(nil)
	_ common.HeadNotifier = (*cliex.HeadSub)(nil)
)

// MgoMetaMgr provides a MetaMgr as the common.MetaManager
func MgoMetaMgr(ctx GlobalContext, dsn MgoMetaMgrDSN) (common.MetaManager, error) {
	mgr, err := mmetamgr.New(ctx, string(dsn))
	return mgr, err
}

// MgoHeadNotifier provides a common.HeadNotifier based on metads
func MgoHeadNotifier(cli api.FullNode) (common.HeadNotifier, error) {
	sub, err := cliex.NewHeadSub(cli)
	return sub, err
}

type raIn struct {
	fx.In
	Ctx     GlobalContext
	Cfg     racailum.Config
	Sub     common.HeadNotifier
	MetaMgr common.MetaManager
	CS      common.ChainStore
	Stm     common.StateManager
	SegOpts []segment.OptionFn `group:"segopt"`
}

// NewRaCailum constructs an instance of RaCailum
func NewRaCailum(in raIn) (*racailum.RaCailum, error) {
	return racailum.New(in.Ctx, in.Cfg, in.Sub, in.MetaMgr, in.CS, in.Stm, in.SegOpts...)
}

// SegmentOpt is used to combine a group of option funcs
type SegmentOpt struct {
	fx.Out
	Opt segment.OptionFn `group:"segopt"`
}
