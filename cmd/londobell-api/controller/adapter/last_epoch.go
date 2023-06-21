package adapter

import (
	"context"
	"net/http"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetLastEpochInfo(c *gin.Context) {
	alog := log.With("method", "GetLastEpochInfo")
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	api := fullnode.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch-1), types.EmptyTSK)
	}

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var winCount int64
	for _, b := range ts.Blocks() {
		winCount += b.ElectionProof.WinCount
	}

	pact, err := api.StateGetActor(ctx, power.Address, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

	pst, err := power.Load(stor, pact)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pc, err := pst.TotalPower()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ract, err := api.StateGetActor(ctx, reward.Address, ts.Key())
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

	currentTotalStoragePowerReward, err := rst.TotalStoragePowerReward()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	parentTs, err := api.ChainGetTipSet(ctx, types.NewTipSetKey(ts.Blocks()[0].Parents...))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	parentRoot := parentTs.ParentState()
	parentTree, err := state.LoadStateTree(stor, parentRoot)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pract, err := parentTree.GetActor(reward.Address)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	prst, err := reward.Load(stor, pract)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	parentTotalStoragePowerReward, err := prst.TotalStoragePowerReward()
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	netRewards := big.Sub(currentTotalStoragePowerReward, parentTotalStoragePowerReward)

	resData := model.EpochRes{
		Cids:            ts.Cids(),
		Parents:         ts.Parents(),
		Epoch:           ts.Height(),
		BlockTime:       common.CalcTimeByEpoch(uint64(ts.Height())),
		BlockCount:      len(ts.Blocks()),
		WinCount:        winCount,
		NetPower:        pc.RawBytePower,
		NetQualityPower: pc.QualityAdjPower,
		NetRewards:      netRewards,
		BaseFee:         ts.Blocks()[0].ParentBaseFee,
		Source:          "api",
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
