package aggregators

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/account"
	_init "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/store"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("aggregators")

	RobustMap = make(map[string]string) // ID: robust
	RLock     sync.RWMutex

	AllDealCountMap = make(map[abi.ChainEpoch]int64)
	DLock           sync.RWMutex

	DealsByAddrCountMap = make(map[string]map[abi.ChainEpoch]int64) // ID: {epoch: count}
	DALock              sync.RWMutex
)

// todo: 有无必要加缓存？
func GetIDByAddr(ctx context.Context, addrStr string) (string, error) {
	addr, err := address.NewFromString(buildnet.NetPrefix + addrStr)
	if err != nil {
		return "", err
	}

	switch addr.Protocol() {
	case address.ID:
		return addrStr, nil
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		api := fullnode.API.GetAppropriateAPI()
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return "", err
		}

		return ID.String()[1:], nil
	default:
		err = fmt.Errorf("invalid addr %v", addrStr)
		return "", err
	}
}

// GetAddrs get [ID,robust] from ActorBalance
func GetAddrs(ctx context.Context, addr string) (model.AddressRes, error) {
	formal := multiquery.DBStateManager.GetFormalCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(formal.Url())
	if !ok {
		return model.AddressRes{}, fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
	}

	var addressRes []model.AddressRes

	pipe, err := util.Parse(model.Ctx{Addr: addr}, string(addressAggregator))
	if err != nil {
		return model.AddressRes{}, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "ActorBalance" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return model.AddressRes{}, err
			}

			err = cur.All(ctx, &addressRes)
			if err != nil {
				return model.AddressRes{}, err
			}

			if len(addressRes) != 1 {
				return model.AddressRes{}, ErrNotFound
			}

			return addressRes[0], nil
		}
	}

	return model.AddressRes{}, fmt.Errorf("no table ActorBalance")
}

func GetRobustByID(ctx context.Context, api v0api.FullNode, IDAddr address.Address, actor *types.Actor) (string, error) {
	if IDAddr.Protocol() != address.ID {
		// warn
		return "", nil
	}

	actorID := IDAddr.String()[1:]

	// find from RobustMap
	RLock.RLock()
	robust, ok := RobustMap[actorID]
	RLock.RUnlock()
	if ok {
		return robust, nil
	}

	res, err := GetAddrs(ctx, actorID)
	if err != nil && err != ErrNotFound {
		return "", err
	}

	if err == nil {
		for _, addr := range res.Addresses {
			if addr == "" {
				continue
			}
			if addr[0] == '1' || addr[0] == '2' || addr[0] == '3' {
				RLock.Lock()
				defer RLock.Unlock()
				RobustMap[actorID] = addr
				return addr, nil
			}
		}

		// 该actor没有robust地址
		return "", nil
	}

	// not found，则为近高度新增的actor
	// account
	if builtin.IsAccountActor(actor.Code) {
		st, err := account.Load(store.ActorStore(ctx, blockstore.NewAPIBlockstore(api)), actor)
		if err != nil {
			return "", err
		}

		robust, err := st.PubkeyAddress()
		if err != nil {
			return "", err
		}

		RLock.Lock()
		RobustMap[actorID] = robust.String()[1:]
		RLock.Unlock()

		return robust.String()[1:], nil
	}

	// other
	iact, err := api.StateGetActor(ctx, _init.Address, types.EmptyTSK)
	if err != nil {
		return "", err
	}

	ist, err := _init.Load(store.ActorStore(ctx, blockstore.NewAPIBlockstore(api)), iact)
	if err != nil {
		return "", err
	}

	var robustStr string
	err = ist.ForEachActor(func(id abi.ActorID, robust address.Address) error {
		idAddr, err := address.NewIDAddress(uint64(id))
		if err != nil {
			return err
		}

		if idAddr.String()[1:] == actorID {
			RLock.Lock()
			RobustMap[actorID] = robust.String()[1:]
			RLock.Unlock()

			robustStr = robust.String()[1:]
			return nil
		}

		return fmt.Errorf("id %v not found", actorID)
	})

	if err != nil {
		return "", err
	}

	return robustStr, nil
}

// GetAllAddrs get [ID, robust, delegated]
func GetAllAddrs(ctx context.Context, addrStr string, api v0api.FullNode) ([]string, error) {
	addrs := make([]string, 0)

	addr, err := address.NewFromString(buildnet.NetPrefix + addrStr)
	if err != nil {
		return nil, err
	}

	switch addr.Protocol() {
	case address.ID:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		if actor.Address != nil {
			delegated := actor.Address
			addrs = append(addrs, delegated.String()[1:])
		}

		robust, err := GetRobustByID(ctx, api, addr, actor)
		if err != nil {
			return nil, err
		}

		if robust != "" {
			addrs = append(addrs, robust)
		}

		return addrs, nil
	case address.SECP256K1, address.Actor, address.BLS:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		if actor.Address != nil {
			delegated := actor.Address
			addrs = append(addrs, delegated.String()[1:])
		}

		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		addrs = append(addrs, ID.String()[1:])

		return addrs, nil
	case address.Delegated:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		robust, err := GetRobustByID(ctx, api, ID, actor)
		if err != nil {
			return nil, err
		}

		if robust != "" {
			addrs = append(addrs, robust)
		}

		return addrs, nil
	default:
		return nil, fmt.Errorf("invalid addr: %v", addrStr)
	}
}
