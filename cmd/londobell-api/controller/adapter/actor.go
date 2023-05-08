package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/filecoin-project/lotus/chain/actors/builtin/datacap"
	"github.com/filecoin-project/lotus/chain/actors/builtin/evm"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/account"
	init_ "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/actors/builtin/market"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	_init "github.com/filecoin-project/lotus/chain/actors/builtin/init"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetActorInfo(c *gin.Context) {
	alog := log.With("method", "GetActorsInfo")
	req := model.ActorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet

	api := API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	addr, err := address.NewFromString(req.ActorID)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	var (
		actorID   address.Address
		actorAddr address.Address
		actorType string
		state     interface{}
	)

	// todo: mask protocol details
	if addr.Protocol() == address.ID {
		actorID = addr
	} else if addr.Protocol() == address.BLS || addr.Protocol() == address.SECP256K1 || addr.Protocol() == address.Actor || addr.Protocol() == address.Delegated {
		actorID, err = api.StateLookupID(ctx, addr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		actorAddr = addr
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

	act, err := api.StateGetActor(ctx, addr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	switch {
	case builtin.IsAccountActor(act.Code):
		actorType = "account"
		st, err := account.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()

		actorAddr, err = api.StateAccountKey(ctx, addr, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	case builtin.IsMultisigActor(act.Code):
		actorType = "multisig"
		st, err := multisig.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsStoragePowerActor(act.Code):
		actorType = "power"
		st, err := power.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsRewardActor(act.Code):
		actorType = "reward"
		st, err := reward.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsInitActor(act.Code):
		actorType = "init"
		st, err := init_.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsStorageMarketActor(act.Code):
		actorType = "market"
		st, err := market.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsVerifiedRegistryActor(act.Code):
		actorType = "verify"
		st, err := verifreg.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsSystemActor(act.Code):
		//system没有state？？
		actorType = "system"
		st, err := MakeSystemState(stor, act.Code)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case builtin.IsStorageMinerActor(act.Code):
		actorType = "miner"
		st, err := miner.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case builtin.IsPaymentChannelActor(act.Code):
		actorType = "paych"
		st, err := paych.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	//case IsBurntFundsActor(addr):
	//	actorType = "burnt"
	case IsCronActor(addr):
		actorType = "cron"
	case IsDataCapActor(addr):
		actorType = "datacap"
		st, err := datacap.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case builtin.IsEvmActor(act.Code):
		actorType = "evm"
		// todo: f2
		if addr.Protocol() == address.ID {
			actorAddr, err = api.StateAccountKey(ctx, addr, ts.Key())
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
		}

		st, err := evm.Load(stor, act)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		state = st.GetState()
	case IsEamActor(addr):
		actorType = "eam"
	case builtin.IsEthAccountActor(act.Code):
		actorType = "ethaccount"
		// todo: f2
		if addr.Protocol() == address.ID {
			actorAddr, err = api.StateAccountKey(ctx, addr, ts.Key())
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
		}
	case builtin.IsPlaceholderActor(act.Code):
		// todo: f2
		actorType = "placeholder"
		if addr.Protocol() == address.ID {
			actorAddr, err = api.StateAccountKey(ctx, addr, ts.Key())
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
		}
	default:
		util.ReturnOnErr(c, alog, fmt.Errorf("unknow actor type: %v", addr))
		return
	}

	resData := model.ActorRes{
		ActorID:   actorID,
		ActorAddr: actorAddr.String(),
		Epoch:     ts.Height(),
		BlockTime: common.CalcTimeByEpoch(uint64(ts.Height())),
		ActorType: actorType,
		Balance:   act.Balance,
		Code:      act.Code,
		Head:      act.Head,
		Nonce:     act.Nonce,
		State:     state,
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)

	alog.Infof("begin test...")
	// test
	nts, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(2830320), types.EmptyTSK)
	if err != nil {
		return
	}
	iact, err := api.StateGetActor(ctx, _init.Address, nts.Key())
	if err != nil {
		return
	}

	ist, err := _init.Load(stor, iact)
	if err != nil {
		return
	}

	robustMap := make(map[address.Address]address.Address)
	err = ist.ForEachActor(func(id abi.ActorID, addr address.Address) error {
		idAddr, err := address.NewIDAddress(uint64(id))
		if err != nil {
			return fmt.Errorf("failed to write to addr map: %w", err)
		}

		robustMap[idAddr] = addr

		return nil
	})

	if err != nil {
		return
	}

	actors, err := api.StateListActors(ctx, nts.Key())
	if err != nil {
		return
	}

	alog.Infof("being write...")
	file, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/ist.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return
	}
	defer file.Close()

	tfile, err := os.OpenFile("/Users/zhoulin/londobell/cmd/londobell-api/aggregators/tree.txt", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return
	}
	defer tfile.Close()

	for addr := range robustMap {
		_, err = io.WriteString(file, addr.String())
		if err != nil {
			return
		}

		_, err = io.WriteString(file, "\n")
		if err != nil {
			return
		}
	}

	for _, addr := range actors {
		_, err = io.WriteString(tfile, addr.String())
		if err != nil {
			return
		}

		_, err = io.WriteString(tfile, "\n")
		if err != nil {
			return
		}
	}

	alog.Info(len(robustMap), len(actors))

}
