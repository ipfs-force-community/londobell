package multiquery

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	levelds "github.com/ipfs/go-ds-leveldb"
	metricsi "github.com/ipfs/go-metrics-interface"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
)

type GlobalContext context.Context
type SegmentMetaDS dtypes.MetadataDS
type RepoPath string

func DBStateManagerOption() dix.Option {
	return dix.Options(
		dix.Override(new(Config), LoadRaConfig),
		dix.Override(new(*DataBaseStateCache), NewDataBaseStateCache),
		dix.Override(new(*DBCollectionsConfigMgr), NewDBCollectionsConfigMgr),
		SegmentManager(),
		dix.Override(new(*Segment), NewSegment))
}

func SegmentManager() dix.Option {
	return dix.Options(
		//dix.Override(new(Config), LoadRaConfig),
		dix.Override(new(SegmentMetaDS), OpenSegmentDS),
		dix.Override(new(*StateManager), NewStateManagerDS))
}

func OpenSegmentDS(rpath RepoPath) (SegmentMetaDS, error) {
	return levelDs(SegmentMetaDSPath(rpath), false)
}

func levelDs(path string, readonly bool) (dtypes.MetadataDS, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
		ReadOnly:    readonly,
	})
}

func SegmentMetaDSPath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "state")
}

func NewStateManagerDS(segds SegmentMetaDS) (*StateManager, error) {
	return NewStateManager(segds)
}

const (
	invokeNone dix.Invoke = iota // nolint: varcheck,deadcode

	invokePopulate
)

var (
	RepoFlag = &cli.StringFlag{
		Name:  "repo",
		Usage: "repo path for bell",
		Value: "~/.multi",
	}
)

func ContextModule(ctx context.Context) dix.Option {
	return dix.Options(dix.Override(new(GlobalContext), ctx),
		dix.Override(new(*http.ServeMux), http.NewServeMux()),
		dix.Override(new(helpers.MetricsCtx), metricsi.CtxScope(ctx, "multi-query")))
}

func MultiQuery(ctx context.Context, target ...interface{}) dix.Option {
	return dix.Options(
		ContextModule(ctx),

		dix.If(len(target) > 0, dix.Populate(invokePopulate, target...)),

		//DBStateManagerOption(),

		dix.Override(new(Config), LoadRaConfig),
		dix.Override(new(*DataBaseStateCache), NewDataBaseStateCache),
		dix.Override(new(*DBCollectionsConfigMgr), NewDBCollectionsConfigMgr),
		SegmentManager(),
		dix.Override(new(*Segment), NewSegment),
	)
}

func GetRepoPath(cctx *cli.Context) (RepoPath, error) {
	dir, err := homedir.Expand(cctx.String(RepoFlag.Name))
	if err != nil {
		return "", fmt.Errorf("expand homedir: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir at %s: %w", dir, err)
	}

	return RepoPath(dir), nil
}

func InjectRepoPath(cctx *cli.Context) dix.Option {
	return dix.Override(new(RepoPath), func() (RepoPath, error) {
		return GetRepoPath(cctx)
	})
}
