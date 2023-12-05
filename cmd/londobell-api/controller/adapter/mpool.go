package adapter

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	lbuiltin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs-force-community/londobell/racailum/segment/actor"
)

var ActorReg = consensus.NewActorRegistry()

func GetPendingMessages(c *gin.Context) {
	alog := log.With("method", "GetPendingMessages")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	ts, err := api.ChainHead(ctx)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	msgs, err := api.MpoolPending(ctx, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	if req.Cid != "" {
		mcid, err := cid.Decode(req.Cid)
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		for _, msg := range msgs {
			if msg.Cid().Equals(mcid) || msg.Message.Cid().Equals(mcid) {
				methodName, err := GetMethodName(ctx, alog, api, msg, ts)
				if err != nil && err != util.ErrNotFound {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}

				hash, err := NewEthHashFromSignedMessage(ctx, msg, api)
				if err != nil {
					alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
					util.ReturnOnErr(c, err)
					return
				}

				res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
					Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: ts.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
				}}}
				c.JSON(http.StatusOK, res)
				return
			}
		}

		c.JSON(http.StatusOK, res)
		return
	}

	if req.Hash != "" {
		var g multierror.Group

		for i := range msgs {
			i := i
			msg := msgs[i]
			g.Go(func() error {
				hash, err := NewEthHashFromSignedMessage(ctx, msg, api)
				if err != nil {
					alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
					return err
				}

				if req.Hash == hash.String() {
					methodName, err := GetMethodName(ctx, alog, api, msg, ts)
					if err != nil && err != util.ErrNotFound {
						alog.Error(err)
						return err
					}

					res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
						Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: ts.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
					}}}

					return nil
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		c.JSON(http.StatusOK, res)
		return
	}

	var (
		pendingMessages []model.PendingMessage
		totalCount      int64
		g               multierror.Group
		mutex           sync.Mutex
	)

	for i := range msgs {
		msg := msgs[i]
		g.Go(func() error {
			methodName, err := GetMethodName(ctx, alog, api, msg, ts)
			if err != nil && err != util.ErrNotFound {
				return err
			}

			// todo: InvokeEVM和其他未定义方法区分开??
			// filter by methodName
			if req.MethodName != "" && methodName == req.MethodName {
				hash, err := NewEthHashFromSignedMessage(ctx, msg, api)
				if err != nil {
					return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid())
				}

				mutex.Lock()
				pendingMessages = append(pendingMessages, model.PendingMessage{
					Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: ts.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
				})
				totalCount++
				mutex.Unlock()
			}

			if req.MethodName == "" {
				hash, err := NewEthHashFromSignedMessage(ctx, msg, api)
				if err != nil {
					return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid())
				}

				// todo: methodName为空时加入mpool吗？
				mutex.Lock()
				pendingMessages = append(pendingMessages, model.PendingMessage{
					Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: ts.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
				})
				totalCount++
				mutex.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	// sort
	sort.Slice(pendingMessages, func(i, j int) bool {
		return pendingMessages[i].Epoch > pendingMessages[j].Epoch
	})

	if req.Index == 0 && req.Limit == 0 {
		res.Data = model.PendingMessagesRes{TotalCount: totalCount, PendingMessages: pendingMessages}
		c.JSON(http.StatusOK, res)
		return
	}

	// paging
	if req.Index*req.Limit >= int64(len(pendingMessages)) {
		c.JSON(http.StatusOK, res)
		return
	}

	if (req.Index+1)*req.Limit >= int64(len(pendingMessages)) {
		res.Data = model.PendingMessagesRes{TotalCount: totalCount, PendingMessages: pendingMessages[req.Index*req.Limit:]}
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = model.PendingMessagesRes{TotalCount: totalCount, PendingMessages: pendingMessages[req.Index*req.Limit : (req.Index+1)*req.Limit]}
	c.JSON(http.StatusOK, res)
}

func GetMethodName(ctx context.Context, log *zap.SugaredLogger, api v0api.FullNode, msg *types.SignedMessage, ts *types.TipSet) (string, error) {
	if msg.Message.Method == abi.MethodNum(0) {
		return "Send", nil
	}

	act, err := api.StateGetActor(ctx, msg.Message.To, ts.Key())
	if err != nil {
		if strings.Contains(err.Error(), "resolution lookup failed") {
			log.Warnf("resolution lookup failed (%s) for message cid(%s) & signedCid(%s): %v\", addr, err", msg.Message.To.String(), msg.Message.Cid().String(), msg.Cid().String(), err)
			return "", nil
		}

		return "", util.ErrNotFound
	}

	code := act.Code
	actorName := lbuiltin.ActorNameByCode(code)

	if ccode, cname, ok := actor.DefaultActorConvertor(ts.Height(), actorName); ok {
		code = ccode
		actorName = cname
	}

	if strings.Contains(actorName, "placeholder") {
		return "Send(placeholder)", nil
	}

	mi, ok := ActorReg.Methods[code][msg.Message.Method]
	if !ok {
		log.Warnf("lookup method for cid=%s, from=%s, to=%s, code=%s, actorName=%s, meth=%d", msg.Message.Cid().String(), msg.Message.From, msg.Message.To, code, actorName, msg.Message.Method)
	}

	return mi.Name, nil
}
