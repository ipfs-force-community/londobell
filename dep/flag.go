package dep

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/lib/cliex"
	"github.com/marmotedu/log"

	"github.com/dtynn/dix"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
)

// common flags
var (
	FullNodeAPIFlag = &cli.StringFlag{
		Name: "api-url",
	}

	TokenFlag = &cli.StringFlag{
		Name: "token",
	}

	RepoFlag = &cli.StringFlag{
		Name:  "bell-repo",
		Usage: "repo path for bell",
		Value: "~/.bell",
	}

	OfflineChainStorageRepoFlag = &cli.StringFlag{
		Name:  "chain-storage-repo",
		Usage: "repo path of chain storage",
		Value: "~/.lotus",
	}
)

func InjectFullNode(cctx *cli.Context) dix.Option {
	return dix.Override(new(v0api.FullNode), func(lc fx.Lifecycle) (v0api.FullNode, error) {
		var requestHeader http.Header
		token := cctx.String("token")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		full, closer, err := client.NewFullNodeRPCV0(cctx.Context, cctx.String("api-url"), requestHeader)
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

func InjectCluster(cctx *cli.Context) dix.Option {
	return dix.Override(new(cliex.Cluster), func(lc fx.Lifecycle) (cliex.Cluster, error) {
		var cluster cliex.Cluster
		token := cctx.String("token")
		api := cctx.String("api-url")
		gapLimit := cctx.Int("gap-limit")
		nodeConfig := cctx.String("nodeconfig")
		if gapLimit == 0 {
			gapLimit = 4
		}
		if nodeConfig != "" {
			if err := util.ParseNodes(nodeConfig); err != nil {
				return cluster, nil
			}
			for _, node := range util.Nodes {
				n := cliex.Node{
					API:   node.URL,
					Token: node.Token,
				}
				cluster.Nodes = append(cluster.Nodes, &n)
			}

		}

		node, err := cliex.InjectFullNode(api, token)
		if err != nil {
			log.Errorf("InjectCluster failed,node: %s err: %s", api, err.Error())
			return cluster, err
		}
		cluster.Current = node
		cluster.Master = api
		cluster.MasterGapLimit = abi.ChainEpoch(gapLimit)
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				cluster.Current.Closer()
				return nil
			},
		})

		return cluster, nil
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

func InjectChainRepo(cctx *cli.Context) dix.Option {
	return dix.Override(new(repo.LockedRepo), func(lc fx.Lifecycle) repo.LockedRepo {
		r, err := repo.NewFS(cctx.String(OfflineChainStorageRepoFlag.Name))
		if err != nil {
			panic(fmt.Errorf("opening fs repo: %w", err))
		}
		exist, err := r.Exists()
		if err != nil {
			panic(fmt.Errorf("check chain store repo exist failed: %w", err))
		}

		if !exist {
			panic(fmt.Errorf("chain store repo not exist,check the path %s", cctx.String(OfflineChainStorageRepoFlag.Name)))
		}
		lr, err := r.Lock(repo.FullNode)
		if err != nil {
			panic(fmt.Errorf("lock repo failed: %w", err))
		}

		return modules.LockedRepo(lr)(lc)
	})
}

func GetWritableOffline(cctx *cli.Context) WritableOffline {
	return WritableOffline(cctx.Bool("writableOffline"))
}

func InjectWritableOffline(cctx *cli.Context) dix.Option {
	return dix.Override(new(WritableOffline), func() WritableOffline {
		return GetWritableOffline(cctx)
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
