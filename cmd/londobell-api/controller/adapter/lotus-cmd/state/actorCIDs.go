package state

import (
	"context"
	"net/http"

	"github.com/filecoin-project/lotus/chain/actors"

	actorstypes "github.com/filecoin-project/go-state-types/actors"

	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateActorCIDs(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateActorCIDs")
	req := lotusCmdModel.StateActorCIDsReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := adapter.API.GetAppropriateAPI()

	var nv network.Version
	if req.NetworkVersion != 0 {
		nv = network.Version(req.NetworkVersion)
	} else {
		nv, err = api.StateNetworkVersion(ctx, types.EmptyTSK)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}
	}

	actorVersion, err := actorstypes.VersionForNetwork(nv)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	manifestCid, _ := actors.GetManifest(actorVersion)

	actorsCids, err := api.StateActorCodeCIDs(ctx, nv)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = lotusCmdModel.StateActorCIDsRes{
		NetworkVersion: nv,
		ActorVersion:   actorVersion,
		ManifestCid:    manifestCid,
		ActorsCids:     actorsCids,
	}

	c.JSON(http.StatusOK, res)
}
