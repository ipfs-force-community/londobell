package aggregators

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/gin-gonic/gin"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

func GetTraceForMessage(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetTraceForMessage")
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

	pipe, err := util.Parse(model.Ctx{Cid: req.Cid}, string(common.TraceForMessageAggregator))
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	var traceForMessageRes []model.TraceForMessageRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
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

		err = json.Unmarshal(rawByte, &traceForMessageRes)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}
	}

	if len(traceForMessageRes) == 0 {
		c.JSON(http.StatusOK, res)
		return
	}

	trace := traceForMessageRes[0]
	err = actorSet.ParseParamsAndReturnsReadable(ctx, &trace)
	if err != nil {
		log.Warnf("ParseParamsAndReturnsReadable failed: %v", err)
	}

	res.Data = []model.TraceForMessageRes{trace}
	c.JSON(http.StatusOK, res)
}

type ActorSet struct {
	m      map[string]cid.Cid
	loadmu sync.RWMutex
}

func NewActorSet() *ActorSet {
	m := make(map[string]cid.Cid)
	return &ActorSet{m: m}
}

var actorSet = NewActorSet()

// 增值服务，错了就用原生的
// todo: 升级关注
func (s *ActorSet) ParseParamsAndReturnsReadable(ctx context.Context, trace *model.TraceForMessageRes) error {
	code := cid.Undef

	toActorID, err := common.GetIDByAddr(ctx, trace.To)
	if err != nil {
		return err
	}

	s.loadmu.RLock()
	found, ok := s.m[toActorID]
	s.loadmu.RUnlock()

	if ok {
		code = found
	}

	toActor, err := address.NewFromString(buildnet.NetPrefix + toActorID)
	if err != nil {
		return err
	}

	if code == cid.Undef {
		api := fullnode.API.GetAppropriateAPI()

		act, err := api.StateGetActor(ctx, toActor, types.EmptyTSK)
		if err != nil {
			return err
		}

		s.loadmu.Lock()
		s.m[toActorID] = act.Code
		s.loadmu.Unlock()

		code = act.Code
	}

	if ccode, _, ok := actor.DefaultActorConvertor(trace.Epoch, lbuiltin.ActorNameByCode(code)); ok {
		code = ccode
	}

	//vma := filcns.NewActorRegistry()
	//
	////todo: realcode
	//mm, ok := vma.Methods[code][abi.MethodNum(trace.MethodNum)]
	//if !ok {
	//	return fmt.Errorf("actor method not found")
	//}
	//
	//mi := actor.MethodInfo{
	//	Actor:  actorName,
	//	Method: mm,
	//}
	//
	//params := mi.ParamObj()
	//if params != nil && len(trace.ParamsBson.Data) > 0 {
	//	err := params.UnmarshalCBOR(bytes.NewReader(trace.ParamsBson.Data))
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	//returns := mi.ReturnObj()
	//if returns != nil && len(trace.ReturnsBson.Data) > 0 {
	//	err := returns.UnmarshalCBOR(bytes.NewReader(trace.ReturnsBson.Data))
	//	if err != nil {
	//		return err
	//	}
	//}

	var ParamsByte []byte

	if IsCreateExternal(toActor, abi.MethodNum(trace.MethodNum)) || IsInvokeContract(code, abi.MethodNum(trace.MethodNum)) {
		// 兼容$binary
		params, ok := trace.Params.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected type of params")
		}

		binaryParams, ok := params["$binary"].(map[string]interface{})
		if ok {
			binaryParamsStr, ok := binaryParams["base64"].(string)
			if ok {
				ParamsByte, err = base64.StdEncoding.DecodeString(binaryParamsStr)
				if err != nil {
					return err
				}
			}
		} else {
			dataParamsStr, ok := params["Data"].(string)
			if ok {
				ParamsByte, err = base64.StdEncoding.DecodeString(dataParamsStr)
				if err != nil {
					return err
				}
			}
		}
	}

	if IsCreateExternal(toActor, abi.MethodNum(trace.MethodNum)) {
		buffer := bytes.NewBuffer(ParamsByte)
		paramsByte, err := cbg.ReadByteArray(buffer, uint64(len(ParamsByte)))
		if err != nil {
			return err
		}

		trace.ParamsDetail = "0x" + hex.EncodeToString(paramsByte)

		var result = &eam.CreateExternalReturn{}
		if len(trace.ReturnsBson.Data) > 0 {
			err := result.UnmarshalCBOR(bytes.NewReader(trace.ReturnsBson.Data))
			if err != nil {
				return err
			}
		}

		var returnDetail struct {
			ActorID       uint64
			RobustAddress string
			EthAddress    string
		}

		ea, err := ethtypes.CastEthAddress(result.EthAddress[:])
		if err != nil {
			return fmt.Errorf("failed to create ethereum address: %w", err)
		}

		returnDetail.ActorID = result.ActorID
		returnDetail.RobustAddress = result.RobustAddress.String()
		returnDetail.EthAddress = ea.String()

		trace.ReturnDetail = returnDetail

		return nil
	}

	if IsInvokeContract(code, abi.MethodNum(trace.MethodNum)) {
		buffer := bytes.NewBuffer(ParamsByte)
		paramsByte, err := cbg.ReadByteArray(buffer, uint64(len(ParamsByte)))
		if err != nil {
			return err
		}

		trace.ParamsDetail = "0x" + hex.EncodeToString(paramsByte)

		result, err := cbg.ReadByteArray(bytes.NewBuffer(trace.ReturnsBson.Data), uint64(len(trace.ReturnsBson.Data)))
		if err != nil {
			return xerrors.Errorf("evm result not correctly encoded: %w", err)
		}

		trace.ReturnDetail = "0x" + hex.EncodeToString(result)

		return nil
	}

	// ExtendSectorExpiration
	//ExtendSectorExpiration2

	return nil
}

func IsCreateExternal(toActor address.Address, methodNum abi.MethodNum) bool {
	return toActor == builtin.EthereumAddressManagerActorAddr && methodNum == builtin.MethodsEAM.CreateExternal
}

func IsInvokeContract(code cid.Cid, methodNum abi.MethodNum) bool {
	return lbuiltin.IsEvmActor(code) && methodNum == builtin.MethodsEVM.InvokeContract
}
