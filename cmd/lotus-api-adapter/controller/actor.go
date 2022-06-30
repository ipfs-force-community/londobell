package controller

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

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetActorInfo(c *gin.Context) {
	req := model.ActorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetActorInfo] bind json ActorReq err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	if req.Epoch == 0 {
		ts, err = API.ChainHead(ctx)
		if err != nil {
			log.Errorf("[GetActorInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetActorInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	addr, err := address.NewFromString(req.ActorID)
	if err != nil {
		log.Errorf("[GetActorInfo] NewFromString err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	var actorID address.Address
	var actorAddr address.Address

	if addr.Protocol() == address.ID {
		actorID = addr
		actorAddr, err = API.StateAccountKey(ctx, addr, ts.Key())
		if err != nil {
			log.Errorf("[GetActorInfo] api StateAccountKey err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else if addr.Protocol() == address.BLS || addr.Protocol() == address.SECP256K1 {
		actorID, err = API.StateLookupID(ctx, addr, ts.Key())
		if err != nil {
			log.Errorf("[GetActorInfo] api StateLookupID err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		actorAddr = addr
	}

	var actorType string

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(API))

	act, err := API.StateGetActor(ctx, addr, ts.Key())
	if err != nil {
		log.Errorf("[GetActorInfo] api StateGetActor err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	var state interface{}

	switch {
	case builtin.IsAccountActor(act.Code):
		actorType = "account"
		st, err := account.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] account.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case builtin.IsMultisigActor(act.Code):
		actorType = "multisig"
		st, err := multisig.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] multisig.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsStoragePowerActor(act.Code):
		actorType = "power"
		st, err := power.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] power.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsRewardActor(act.Code):
		actorType = "reward"
		st, err := reward.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] reward.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsInitActor(act.Code):
		actorType = "init"
		st, err := init_.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] init_.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsStorageMarketActor(act.Code):
		actorType = "market"
		st, err := market.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] market.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsVerifiedRegistryActor(act.Code):
		actorType = "verify"
		st, err := verifreg.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] verifreg.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case IsSystemActor(act.Code):
		//system没有state？？
		actorType = "system"
		st, err := MakeSystemState(stor, act.Code)
		if err != nil {
			log.Errorf("[GetActorInfo] MakeSystemState err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case builtin.IsStorageMinerActor(act.Code):
		actorType = "miner"
		st, err := miner.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] miner.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		state = st.GetState()
	case builtin.IsPaymentChannelActor(act.Code):
		actorType = "paych"
		st, err := paych.Load(stor, act)
		if err != nil {
			log.Errorf("[GetActorInfo] paych.Load err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
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
		BlockTime: CalcTimeByEpoch(uint64(ts.Height())),
		ActorType: actorType,
		Balance:   act.Balance,
		Code:      act.Code,
		Head:      act.Head,
		State:     state,
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
