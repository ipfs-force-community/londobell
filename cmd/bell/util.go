package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dtynn/dix"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/metrics"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/ipfs-force-community/londobell/api"
	"github.com/ipfs-force-community/londobell/dep"
	"github.com/ipfs-force-community/londobell/racailum"
)

var (
	log   = logging.Logger("bell")
	fxlog = &fxLogger{
		ZapEventLogger: log,
	}
)

type fxLogger struct {
	*logging.ZapEventLogger
}

// Printf impls fx.Printer.Printf
func (l *fxLogger) Printf(msg string, args ...interface{}) {
	l.ZapEventLogger.Debugf(msg, args...)
}

func parsetTipSetKey(s string) (types.TipSetKey, error) {
	cids, err := lcli.ParseTipSetString(s)
	if err != nil {
		return types.EmptyTSK, err
	}

	return types.NewTipSetKey(cids...), nil
}

func GetAPIV0(ctx *cli.Context) (api.BellAPI, jsonrpc.ClientCloser, error) {
	var res api.BellAPIStruct
	rpath, err := dep.GetRepoPath(ctx)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := dep.LoadRaConfig(rpath)
	if err != nil {
		return nil, nil, err
	}
	muladdr := cfg.HTTP.RPCListen
	if muladdr == "" {
		muladdr = racailum.DefaultRPCListenAddr
	}
	addr, err := cliutil.APIInfo{Addr: muladdr}.DialArgs("v0")
	if err != nil {
		return nil, nil, err
	}
	closer, err := jsonrpc.NewMergeClient(ctx.Context, addr, "Bell",
		[]interface{}{
			&res.Internal,
		},
		nil,
	)
	return &res, closer, err
}

func ServeRPC(a api.BellAPI, stop dix.StopFunc, addr multiaddr.Multiaddr, shutdownCh <-chan struct{}, maxRequestSize int64) error {
	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 { // config set
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}
	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register("Bell", a)

	http.Handle("/rpc/v0", rpcServer)
	lst, err := manet.Listen(addr)
	if err != nil {
		return xerrors.Errorf("could not listen: %w", err)
	}

	srv := &http.Server{
		Handler: http.DefaultServeMux,
		BaseContext: func(listener net.Listener) context.Context {
			ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.APIInterface, "lotus-daemon"))
			return ctx
		},
	}

	sigCh := make(chan os.Signal, 2)
	shutdownDone := make(chan struct{})
	go func() {
		select {
		case sig := <-sigCh:
			log.Warnw("received shutdown", "signal", sig)
		case <-shutdownCh:
			log.Warn("received shutdown")
		}

		log.Warn("Shutting down...")
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
		if err := stop(context.TODO()); err != nil {
			log.Errorf("graceful shutting down failed: %s", err)
		}
		log.Warn("Graceful shutdown successful")
		_ = log.Sync() //nolint:errcheck
		close(shutdownDone)
	}()
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	err = srv.Serve(manet.NetListener(lst))
	if err == http.ErrServerClosed {
		<-shutdownDone
		return nil
	}
	return err
}
