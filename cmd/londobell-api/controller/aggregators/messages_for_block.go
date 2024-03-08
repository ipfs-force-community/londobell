package aggregators

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"

	"context"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/common"
)

var query = `
[
	{
		"$match":{
			IsBlock : true,
			Type : "from",
			Epoch : %d,
			$or: [
				{Cid:{$in: %s}},
				{SignedCid:{$in: %s}}
			],				
		}
	},
	{
		$skip: ctx.Skip
	},
	{
		$limit: ctx.Limit
	},	
	{
		$project: {
			Cid: "$Cid",
			Epoch: "$Epoch",
			Value: "$Value",
			From: "$From",
			To: "$To",
			ExitCode: "$ExitCode",
			Method: "$MethodName",
		}
	}	
]
`

func GetMessagesForBlock(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMessagesForBlock")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if req.Index == 0 && req.Limit == 0 {
		req.Limit = math.MaxInt64
	}

	// get block message
	messagesForBlockRes, err := getBlockMsg(req, countUtils)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = messagesForBlockRes
	c.JSON(http.StatusOK, res)
}

type BlockMessage struct {
	// Cid      cid.Cid
	Epoch    abi.ChainEpoch
	Messages []string `bson:"Messages"`
}

func getBlockMsg(req model.CommonReq, countUtils []multiquery.CountUtil) ([]model.TraceForMessageSimplifyRes, error) {
	var (
		blockMsgs []BlockMessage
		res       []model.TraceForMessageSimplifyRes
	)
	script := `
[    
	{
        $match: {
            _id: ctx.Cid,
        }
    }
]	
	`

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipe, err := util.Parse(model.Ctx{Cid: req.Cid}, script)
	if err != nil {
		return res, err
	}
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "BlockMessage")
		if err != nil {
			return res, err
		}

		if len(multiResult) == 0 {
			return res, err
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return res, err
		}

		err = json.Unmarshal(rawByte, &blockMsgs)
		if err != nil {
			return res, err
		}
	}

	if err != nil {
		return res, err
	}

	if len(blockMsgs) == 0 {
		return res, err
	}
	cids := formatList(blockMsgs[0].Messages)
	script = fmt.Sprintf(query, blockMsgs[0].Epoch, cids, cids)
	pipe, err = util.Parse(model.Ctx{Cid: req.Cid, Skip: req.Index * req.Limit, Limit: req.Limit}, script)

	if err != nil {
		return res, err
	}

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ActorMessage")
		if err != nil {
			return res, err
		}

		if len(multiResult) == 0 {
			return res, nil
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return res, err
		}

		err = json.Unmarshal(rawByte, &res)
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

func formatList(list []string) string {
	jsonBytes, err := json.Marshal(list)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return string(jsonBytes)

}
