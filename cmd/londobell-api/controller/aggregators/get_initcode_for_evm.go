package aggregators

import (
	"encoding/json"
	"net/http"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetInitCodeForEvm(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetInitCodeForEvm")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	req.Addr, err = GetIDByAddr(ctx, req.Addr)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pipe, err := util.Parse(model.Ctx{Addr: req.Addr}, monitor.GetGetEvminitcodeByActorIDAggregator())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var evmInitCodeByActorIDRes []model.InitCodeForEvmRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "EvmInitCode")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if len(multiResult) == 0 {
			c.JSON(http.StatusOK, res)
			return
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		err = json.Unmarshal(rawByte, &evmInitCodeByActorIDRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	if len(evmInitCodeByActorIDRes) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = evmInitCodeByActorIDRes[0]
	c.JSON(http.StatusOK, res)

	//// 方案二
	//// 不处理0x地址
	//IDStr, err := GetIDByAddr(ctx, req.Addr)
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//actorID, err := address.NewFromString(buildnet.NetPrefix + IDStr)
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//ID, err := address.IDFromAddress(actorID)
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//pipe, err := util.Parse(model.Ctx{ID: ID}, monitor.GetInitCodeForEvmAggregator())
	//if err != nil {
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//var initCodeForEvm []model.InitCodeForEvm
	//
	//// multi dbs query
	//{
	//	multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
	//	if err != nil {
	//		alog.Error(err)
	//		util.ReturnOnErr(c, err)
	//		return
	//	}
	//
	//	if len(multiResult) == 0 {
	//		c.JSON(http.StatusOK, res)
	//		return
	//	}
	//
	//	raw := multiResult
	//	rawByte, err := json.Marshal(raw)
	//	if err != nil {
	//		alog.Error(err)
	//		util.ReturnOnErr(c, err)
	//		return
	//	}
	//
	//	err = json.Unmarshal(rawByte, &initCodeForEvm)
	//	if err != nil {
	//		alog.Error(err)
	//		util.ReturnOnErr(c, err)
	//		return
	//	}
	//}
	//
	//if len(initCodeForEvm) == 0 {
	//	c.JSON(http.StatusOK, res)
	//	return
	//}
	//
	//// 兼容$binary
	//var ParamsByte []byte
	//params, ok := initCodeForEvm[0].InitCode.(map[string]interface{})
	//if !ok {
	//	err = fmt.Errorf("unexpected type of params")
	//	alog.Error(err)
	//	util.ReturnOnErr(c, err)
	//	return
	//}
	//
	//binaryParams, ok := params["$binary"].(map[string]interface{})
	//if ok {
	//	binaryParamsStr, ok := binaryParams["base64"].(string)
	//	if ok {
	//		ParamsByte, err = base64.StdEncoding.DecodeString(binaryParamsStr)
	//		if err != nil {
	//			alog.Error(err)
	//			util.ReturnOnErr(c, err)
	//			return
	//		}
	//	}
	//} else {
	//	dataParamsStr, ok := params["Data"].(string)
	//	if ok {
	//		ParamsByte, err = base64.StdEncoding.DecodeString(dataParamsStr)
	//		if err != nil {
	//			alog.Error(err)
	//			util.ReturnOnErr(c, err)
	//			return
	//		}
	//	}
	//}
	//
	//var initCode abi.CborBytes
	//err = initCode.UnmarshalCBOR(bytes.NewReader(ParamsByte))
	//if err != nil {
	//	c.JSON(http.StatusOK, res)
	//	return
	//}
	//
	//res.Data = model.InitCodeForEvmRes{InitCode: hex.EncodeToString(initCode)}
	//
	//c.JSON(http.StatusOK, res)
}
