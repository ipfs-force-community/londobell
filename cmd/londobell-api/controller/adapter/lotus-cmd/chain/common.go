package chain

import (
	"bytes"
	"context"
	"fmt"

	lapi "github.com/filecoin-project/lotus/api"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
)

type apiIpldStore struct {
	ctx context.Context
	api v0api.FullNode
}

func (ht *apiIpldStore) Context() context.Context {
	return ht.ctx
}

func (ht *apiIpldStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	raw, err := ht.api.ChainReadObj(ctx, c)
	if err != nil {
		return err
	}

	cu, ok := out.(cbg.CBORUnmarshaler)
	if ok {
		if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("Object does not implement CBORUnmarshaler")
}

func (ht *apiIpldStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	panic("No mutations allowed")
}

func handleAmt(ctx context.Context, api v0api.FullNode, r cid.Cid) error {
	s := &apiIpldStore{ctx, api}
	mp, err := adt.AsArray(s, r)
	if err != nil {
		return err
	}

	return mp.ForEach(nil, func(key int64) error {
		fmt.Printf("%d\n", key)
		return nil
	})
}

func handleHamtEpoch(ctx context.Context, api v0api.FullNode, r cid.Cid) error {
	s := &apiIpldStore{ctx, api}
	mp, err := adt.AsMap(s, r)
	if err != nil {
		return err
	}

	return mp.ForEach(nil, func(key string) error {
		ik, err := abi.ParseIntKey(key)
		if err != nil {
			return err
		}

		fmt.Printf("%d\n", ik)
		return nil
	})
}

func handleHamtAddress(ctx context.Context, api v0api.FullNode, r cid.Cid) error {
	s := &apiIpldStore{ctx, api}
	mp, err := adt.AsMap(s, r)
	if err != nil {
		return err
	}

	return mp.ForEach(nil, func(key string) error {
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", addr)
		return nil
	})
}

func apiMsgCids(in []lapi.Message) []cid.Cid {
	out := make([]cid.Cid, len(in))
	for k, v := range in {
		out[k] = v.Cid
	}
	return out
}
