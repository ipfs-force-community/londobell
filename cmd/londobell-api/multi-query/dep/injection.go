package dep

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	levelds "github.com/ipfs/go-ds-leveldb"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"go.uber.org/fx"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/common"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	metricsi "github.com/ipfs/go-metrics-interface"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query/segment"
)

type GlobalContext context.Context
type SegmentMetaDS dtypes.MetadataDS
type RepoPath string

const (
	InvokeNone dix.Invoke = iota // nolint: varcheck,deadcode

	InvokePopulate
)

var (
	RepoFlag = &cli.StringFlag{
		Name:  "repo",
		Usage: "repo path for bell",
		Value: "~/.multi",
	}
)

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

func LoadRaConfig(rpath RepoPath) (common.Config, error) {
	cfgPath := ConfigFilePath(rpath)
	cfg := common.DefaultConfig()
	opt := config.SetDefault(func() (interface{}, error) {
		return &cfg, nil
	})
	_, err := config.FromFile(cfgPath, opt) // todo: config不适合当前数据库配置需求
	if err != nil {
		return common.Config{}, fmt.Errorf("read config from file %s: %w", cfgPath, err)
	}

	return cfg, nil
}

func ConfigFilePath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "config")
}

func MultiQuery(ctx context.Context, target ...interface{}) dix.Option {
	return dix.Options(
		ContextModule(ctx),
		dix.If(len(target) > 0, dix.Populate(InvokePopulate, target...)),
		dix.Override(new(*multiquery.DataBaseStateCache), multiquery.NewDataBaseStateCache),
		dix.Override(new(*common.DBCollectionsConfigMgr), common.NewDBCollectionsConfigMgr),
		SegmentManager(),
		dix.Override(new(common.Config), LoadRaConfig),
		dix.Override(new(*segment.Segment), NewSegment),
	)
}

func ContextModule(ctx context.Context) dix.Option {
	return dix.Options(dix.Override(new(GlobalContext), ctx),
		dix.Override(new(*http.ServeMux), http.NewServeMux()),
		dix.Override(new(helpers.MetricsCtx), metricsi.CtxScope(ctx, "multi-query")))
}

type sIn struct {
	fx.In

	Ctx    GlobalContext
	SegMgr *segment.SegManager
	Config common.Config
}

func NewSegment(in sIn) (*segment.Segment, error) {
	return segment.New(in.Ctx, in.SegMgr, in.Config)
}

func SegmentManager() dix.Option {
	return dix.Options(
		//dix.Override(new(Config), LoadRaConfig),
		dix.Override(new(SegmentMetaDS), OpenSegmentDS),
		dix.Override(new(*segment.SegManager), NewStateManagerDS))
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

func NewStateManagerDS(segds SegmentMetaDS) (*segment.SegManager, error) {
	return segment.NewStateManager(segds)
}
