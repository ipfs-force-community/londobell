package chain

//func GetChainEncode(c *gin.Context) {
//	alog := adapter.Log.With("method", "GetChainEncode")
//	req := lotusCmdModel.ChainEncodeReq{}
//	res := model.CommonRes{Code: model.Success}
//	err := c.BindJSON(&req)
//	if err != nil {
//		util.ReturnOnErr(c, alog, err)
//		return
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	if req.Method == "" {
//		util.ReturnOnErr(c, alog, fmt.Errorf("must pass Method number"))
//		return
//	}
//	method, err := strconv.ParseInt(req.Method, 10, 64)
//	if err != nil {
//		util.ReturnOnErr(c, alog, err)
//		return
//	}
//
//	if req.To == "" {
//		util.ReturnOnErr(c, alog, fmt.Errorf("must pass addr"))
//		return
//	}
//
//	to, err := cid.Parse(req.To)
//	if err != nil {
//		util.ReturnOnErr(c, alog, err)
//		return
//	}
//
//	api := adapter.API.GetAppropriateAPI()
//
//	if err != nil {
//		util.ReturnOnErr(c, alog, err)
//		return
//	}
//
//	// todo: GetFullNodeAPIV1
//	var p []byte
//	p, err = api.StateEncodeParams(ctx, to, abi.MethodNum(method), req.Params)
//	if err != nil {
//		util.ReturnOnErr(c, alog, err)
//		return
//	}
//
//	switch req.Encodeing {
//	case "base64", "b64":
//		res.Data = lotusCmdModel.ChainEncodeRes{
//			Params: base64.StdEncoding.EncodeToString(p),
//		}
//	case "hex":
//		res.Data = lotusCmdModel.ChainEncodeRes{
//			Params: hex.EncodeToString(p),
//		}
//	default:
//		util.ReturnOnErr(c, alog, fmt.Errorf("unknown encoding"))
//		return
//	}
//
//	c.JSON(http.StatusOK, res)
//}
