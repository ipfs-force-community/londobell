package state

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"

	lapi "github.com/filecoin-project/lotus/api"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model/lotusCmdModel"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetStateListMessages(c *gin.Context) {
	alog := adapter.Log.With("method", "GetStateListMessages")
	req := lotusCmdModel.StateListMessagesReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var toa, froma address.Address
	if req.To != "" {
		a, err := address.NewFromString(req.To)
		if err != nil {
			util.ReturnOnErr(c, alog, fmt.Errorf("given 'to' address %q was invalid: %w", req.To, err))
			return
		}
		toa = a
	}

	if req.From != "" {
		a, err := address.NewFromString(req.From)
		if err != nil {
			util.ReturnOnErr(c, alog, fmt.Errorf("given 'from' address %q was invalid: %w", req.From, err))
			return
		}
		froma = a
	}

	toh := abi.ChainEpoch(req.ToHeight)

	var ts *types.TipSet
	api := adapter.API.GetAppropriateAPI()

	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	windowSize := abi.ChainEpoch(100)

	var (
		messages    = make([]*types.Message, 0)
		messageCids = make([]cid.Cid, 0)
	)

	cur := ts
	for cur.Height() > toh {
		if ctx.Err() != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		end := toh
		if cur.Height()-windowSize > end {
			end = cur.Height() - windowSize
		}

		msgs, err := api.StateListMessages(ctx, &lapi.MessageMatch{To: toa, From: froma}, cur.Key(), end)
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		for _, cid := range msgs {
			if req.Cids {
				messageCids = append(messageCids, cid)
				continue
			}

			m, err := api.ChainGetMessage(ctx, cid)
			if err != nil {
				util.ReturnOnErr(c, alog, err)
				return
			}
			messages = append(messages, m)
		}

		if end <= 0 {
			break
		}

		next, err := api.ChainGetTipSetByHeight(ctx, end-1, cur.Key())
		if err != nil {
			util.ReturnOnErr(c, alog, err)
			return
		}

		cur = next
	}

	// todo: return value
	res.Data = lotusCmdModel.StateListMessagesRes{
		Messages:    messages,
		MessageCids: messageCids,
	}

	c.JSON(http.StatusOK, res)
}
