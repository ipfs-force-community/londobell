package chain

import (
	"context"
	"net/http"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetChainGasPrices(c *gin.Context) {
	alog := adapter.Log.With("method", "GetChainGasPrices")
	req := lotusCmdModel.ChainGasPricesReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := adapter.API.GetAppropriateAPI()

	gasPricesRes := make([]lotusCmdModel.ChainGasPricesRes, 0)
	nb := []int{1, 2, 3, 5, 10, 20, 50, 100, 300}
	for _, nblocks := range nb {
		addr := builtin.SystemActorAddr // TODO: make real when used in GasEstimateGasPremium

		est, err := api.GasEstimateGasPremium(ctx, uint64(nblocks), addr, 10000, types.EmptyTSK)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		gasPricesRes = append(gasPricesRes, lotusCmdModel.ChainGasPricesRes{NBlocks: nblocks, EstimateGasPremium: est})
	}

	res.Data = gasPricesRes

	c.JSON(http.StatusOK, res)
}
