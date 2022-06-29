package dep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dtynn/dix"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/api/v0api"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

// common flags
var (
	FullNodeAPIFlag = &cli.StringFlag{
		Name: "api-url",
	}

	RepoFlag = &cli.StringFlag{
		Name:  "bell-repo",
		Usage: "repo path for bell",
		Value: "~/.bell",
	}
)

func InjectFullNode(cctx *cli.Context) dix.Option {
	return dix.Override(new(v0api.FullNode), func(lc fx.Lifecycle) (v0api.FullNode, error) {
		full, closer, err := cliutil.GetFullNodeAPI(cctx)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				closer()
				return nil
			},
		})

		return full, nil
	})
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

func ConfigFilePath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "config")
}

func SegmentMetaDSPath(rpath RepoPath) string {
	return filepath.Join(string(rpath), "segment")
}

func NewAfterGenesisSet() dtypes.AfterGenesisSet {
	return dtypes.AfterGenesisSet{}
}
