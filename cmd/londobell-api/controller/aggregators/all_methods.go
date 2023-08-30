package aggregators

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetAllMethods(c *gin.Context) {
	alog := log.With("method", "GetAllMethods")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var allMethods []model.AllMethodsRes
	for _, method := range util.AllMethodList {
		allMethods = append(allMethods, model.AllMethodsRes{MethodName: method, Count: 1})
	}

	res.Data = allMethods
	c.JSON(http.StatusOK, res)
}
