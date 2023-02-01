package adapter

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

func GetActorIDs(c *gin.Context) {
	alog := log.With("method", "GetActorIDs")
	req := model.ActorIDReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		ts          *types.TipSet
		addrs       []address.Address
		actorIDs    []address.Address
		actorIDsRes model.ActorIDRes
	)

	api := API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	addrs, err = api.StateListActors(ctx, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	for _, addr := range addrs {
		var actorID address.Address

		if addr.Protocol() == address.ID {
			actorID = addr
		} else if addr.Protocol() == address.BLS || addr.Protocol() == address.SECP256K1 {
			actorID, err = api.StateLookupID(ctx, addr, ts.Key())
			if err != nil {
				alog.Error(err)
				util.ReturnOnErr(c, err)
				return
			}
		}
		actorIDs = append(actorIDs, actorID)
	}

	actorIDsRes = model.ActorIDRes{
		ActorIDs:  actorIDs,
		Epoch:     ts.Height(),
		BlockTime: common.CalcTimeByEpoch(uint64(ts.Height())),
	}

	res.Data = actorIDsRes
	c.JSON(http.StatusOK, res)
}
