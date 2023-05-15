package main

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc"
	cliutil "github.com/filecoin-project/lotus/cli/util"

	"github.com/ipfs-force-community/londobell/api"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
)

func ColdsIsExists(db multiquery.DB, colds []multiquery.DB) bool {
	var exist bool

	for _, cold := range colds {
		if db.Equals(cold) {
			exist = true
		}
	}

	return exist
}

func CompleteDataBaseState(ctx context.Context, ds *multiquery.DataBaseState, cols multiquery.Collections, limit, interval int) error {
	if err := multiquery.RefreshBlockMsgs(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshBlockMsgsByMethodName(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshActorMsgsByMethodName(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshActorMsgs(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshActorTransferMsgs(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshMinedMsgsMaps(ctx, ds, cols, limit, interval); err != nil {
		return err
	}
	if err := multiquery.RefreshTransfersForLargeAmount(ctx, ds, cols, limit, interval); err != nil {
		return err
	}

	return nil
}

func GetAPIV0(ctx context.Context, muladdr string) (api.MultiAPI, jsonrpc.ClientCloser, error) {
	var res api.MultiAPIStruct

	if muladdr == "" {
		muladdr = multiquery.DefaultRPCListenAddr
	}
	addr, err := cliutil.APIInfo{Addr: muladdr}.DialArgs("v0")
	if err != nil {
		return nil, nil, err
	}
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Multi",
		[]interface{}{
			&res.Internal,
		},
		nil,
	)
	return &res, closer, err
}
