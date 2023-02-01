package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/mongoutil"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"golang.org/x/net/context"
)

// 24hbasefee走势取一天整点数据
func GetTipSet(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTipSet")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var tipSetRes []model.TipSetRes
	pipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch}, string(tipsetAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	cur, err := mongoutil.TipSetCol.Aggregate(ctx, pipe)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	err = cur.All(ctx, &tipSetRes)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// search from the temporary repository if not found
	if len(tipSetRes) == 0 {
		tmpPipe, err := Parse(model.Ctx{StartEpoch: req.StartEpoch}, string(tipsetAggregator))
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		tmpCur, err := mongoutil.TmpTipSetCol.Aggregate(ctx, tmpPipe)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = tmpCur.All(ctx, &tipSetRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	res.Data = tipSetRes
	c.JSON(http.StatusOK, res)
}
