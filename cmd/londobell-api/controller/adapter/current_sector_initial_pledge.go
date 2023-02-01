package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-state-types/big"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/go-state-types/builtin/v9/util/smoothing"

	"github.com/filecoin-project/lotus/chain/stmgr"

	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func CurrentSectorInitialPledge(c *gin.Context) {
	alog := log.With("method", "CurrentSectorInitialPledge")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	head, err := Components.Full.ChainHead(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	sm, ok := Components.SM.(*stmgr.StateManager)
	if !ok {
		err = fmt.Errorf("Components.SM is not *stmgr.StateManager type")
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cs, ok := Components.CS.(*store.ChainStore)
	if !ok {
		err = fmt.Errorf("Components.CS is not *store.ChainStore type")
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	root := head.ParentState()
	tree, err := sm.StateTree(root)
	if err != nil {
		err = fmt.Errorf("load state tree for %s: %w", root, err)
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	supply, err := sm.GetVMCirculatingSupplyDetailed(ctx, head.Height(), tree)
	if err != nil {
		err = fmt.Errorf("get vm circulating supply: %w", err)
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	stor := cs.ActorStore(ctx)
	pact, err := sm.LoadActor(ctx, power.Address, head)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pst, err := power.Load(stor, pact)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	QualityAdjPowerSmoothed, err := pst.TotalPowerSmoothed()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ract, err := sm.LoadActor(ctx, reward.Address, head)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	rst, err := reward.Load(stor, ract)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	thisEpochBaselinePower, err := rst.ThisEpochBaselinePower()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	thisEpochRewardSmoothed, err := rst.ThisEpochRewardSmoothed()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	initPledge := miner9.InitialPledgeForPower(big.MustFromString("1099511627776"), thisEpochBaselinePower, smoothing.FilterEstimate(thisEpochRewardSmoothed), smoothing.FilterEstimate(QualityAdjPowerSmoothed), supply.FilCirculating)
	circulatingf, err := strconv.ParseFloat(supply.FilCirculating.String(), 64)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	totalf, err := strconv.ParseFloat("2000000000000000000000000000", 64)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData := model.CurrentSectorInitialPledgeRes{
		CirculatingRate:            circulatingf / totalf,
		CurrentSectorInitialPledge: initPledge,
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
