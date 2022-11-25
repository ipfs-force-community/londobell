package adapter

import (
	"context"
	"net/http"

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

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetActorsInfo(c *gin.Context) {
	alog := Log.With("method", "GetActorsInfo")
	req := model.ActorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ts        *types.TipSet
		addrs     []address.Address
		actorsRes []model.ActorRes
	)

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

	if req.ActorID == "" {
		addrs, err = api.StateListActors(ctx, ts.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	} else {
		addr, err := address.NewFromString(req.ActorID)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
		addrs = append(addrs, addr)
	}

	for _, addr := range addrs {
		var (
			actorID   address.Address
			actorAddr address.Address
			actorType string
			state     interface{}
		)

		if addr.Protocol() == address.ID {
			actorID = addr
		} else if addr.Protocol() == address.BLS || addr.Protocol() == address.SECP256K1 {
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
		case IsBurntFundsActor(addr):
			actorType = "burnt"
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

		actorsRes = append(actorsRes, resData)
	}

	res.Data = actorsRes
	c.JSON(http.StatusOK, res)
}
