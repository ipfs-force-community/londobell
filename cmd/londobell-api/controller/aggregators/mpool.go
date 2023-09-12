package aggregators

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs/go-cid"
)

func GetMpool(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alog := log.With("method", "GetMpool")
	req := model.CommonReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	api := fullnode.API.GetAppropriateAPI()

	unStoredMsgs, head, err := getUnStoredMsgs(ctx, api)

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

		for cur, msgs := range unStoredMsgs {
			for _, msg := range msgs {
				if msg.Cid().Equals(mcid) || msg.Message.Cid().Equals(mcid) {
					methodName, err := adapter.GetMethodName(ctx, alog, api, msg, cur)
					if err != nil && err != util.ErrNotFound {
						alog.Error(err)
						util.ReturnOnErr(c, err)
						return
					}

					hash, err := adapter.NewEthHashFromSignedMessage(ctx, msg, api)
					if err != nil {
						alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
						util.ReturnOnErr(c, err)
						return
					}

					res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
						Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: cur.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
					}}}
					c.JSON(http.StatusOK, res)
					return
				}
			}
		}

		// 查看是否在数据库中
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

		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
		if err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		// 如果msg不在数据库中且不在unStoredMsgs,且在链上能查到,则作为pending消息返回;
		if len(multiResult) == 0 {
			alog.Warnf("mcid: %s dont exist in db nor unStoredMsgs", mcid)
			msg, _ := api.ChainGetMessage(ctx, mcid)
			var smsg = new(types.SignedMessage)
			if msg != nil {
				smsg.Message = *msg
				methodName, err := adapter.GetMethodName(ctx, alog, api, smsg, head)
				if err != nil && err != util.ErrNotFound {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}

				hash, err := adapter.NewEthHashFromSignedMessage(ctx, smsg, api)
				if err != nil {
					alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
					util.ReturnOnErr(c, err)
					return
				}
				// 当mcid与smsg.Message.Cid()不同时说明消息被覆盖掉了
				if mcid == smsg.Message.Cid() {
					res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
						Cid: smsg.Message.Cid(), SignedCid: msg.Cid(), Epoch: head.Height(), From: smsg.Message.From, To: smsg.Message.To, Value: smsg.Message.Value, GasLimit: smsg.Message.GasLimit, GasPremium: smsg.Message.GasPremium, Method: methodName, Hash: hash.String(),
					}}}
					c.JSON(http.StatusOK, res)
					return
				}

			}

		}

		c.JSON(http.StatusOK, res)
		return
	}

	if req.Hash != "" {
		var g multierror.Group
		for cur, msgs := range unStoredMsgs {
			for i := range msgs {
				cur := cur
				i := i
				msg := msgs[i]
				g.Go(func() error {
					hash, err := adapter.NewEthHashFromSignedMessage(ctx, msg, api)
					if err != nil {
						alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
						return err
					}

					if req.Hash == hash.String() {
						methodName, err := adapter.GetMethodName(ctx, alog, api, msg, cur)
						if err != nil && err != util.ErrNotFound {
							alog.Error(err)
							return err
						}

						res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
							Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: cur.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
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
	}

	var (
		pendingMessages []model.PendingMessage
		totalCount      int64
		g               multierror.Group
		mutex           sync.Mutex
	)
	for cur, msgs := range unStoredMsgs {
		for i := range msgs {
			cur := cur
			msg := msgs[i]
			g.Go(func() error {
				methodName, err := adapter.GetMethodName(ctx, alog, api, msg, cur)
				if err != nil && err != util.ErrNotFound {
					return err
				}

				// todo: InvokeEVM和其他未定义方法区分开??
				// filter by methodName
				if req.MethodName != "" && methodName == req.MethodName {
					hash, err := adapter.NewEthHashFromSignedMessage(ctx, msg, api)
					if err != nil {
						return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid())
					}

					mutex.Lock()
					pendingMessages = append(pendingMessages, model.PendingMessage{
						Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: cur.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
					})
					totalCount++
					mutex.Unlock()
				}

				if req.MethodName == "" {
					hash, err := adapter.NewEthHashFromSignedMessage(ctx, msg, api)
					if err != nil {
						return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid())
					}

					// todo: methodName为空时加入mpool吗？
					mutex.Lock()
					pendingMessages = append(pendingMessages, model.PendingMessage{
						Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: cur.Height(), From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value, GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium, Method: methodName, Hash: hash.String(),
					})
					totalCount++
					mutex.Unlock()
				}

				return nil
			})
		}
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

func getLatestDBHeight(ctx context.Context) (abi.ChainEpoch, error) {
	var Height abi.ChainEpoch
	latestTipSetRes, err := getLatestTipSet(ctx)
	if err != nil {
		return Height, err
	}
	if len(latestTipSetRes) < 1 {
		err := fmt.Errorf("getLatestTipSet return err:  %v", latestTipSetRes)
		return Height, err
	}
	Height = latestTipSetRes[0].Epoch
	return Height, nil
}

func getUnStoredMsgs(ctx context.Context, fullAPI v0api.FullNode) (map[*types.TipSet][]*types.SignedMessage, *types.TipSet, error) {

	mpools := make(map[*types.TipSet][]*types.SignedMessage)
	head, err := fullAPI.ChainHead(ctx)
	if err != nil {
		return nil, nil, err
	}

	dbHeight, err := getLatestDBHeight(ctx)

	if err != nil {
		return mpools, head, err
	}

	pendingMsgs, err := fullAPI.MpoolPending(ctx, head.Key())

	if err != nil {
		return mpools, head, err
	}

	mpools[head] = pendingMsgs
	chainHeight := head.Height()
	currentTS := head
	for ; chainHeight > dbHeight+1; chainHeight-- {

		var tipSetMsgs []*types.SignedMessage
		msgs, err := fullAPI.ChainGetMessagesInTipset(ctx, currentTS.Key())
		if err != nil {
			return mpools, head, err
		}

		for _, m := range msgs {
			var t = new(types.SignedMessage)
			if m.Message != nil {
				t.Message = *m.Message
				tipSetMsgs = append(tipSetMsgs, t)
			}

		}
		if currentTS == head {
			mpools[head] = append(mpools[head], tipSetMsgs...)
		} else {
			mpools[currentTS] = tipSetMsgs
		}

		currentTS, err = fullAPI.ChainGetTipSet(ctx, currentTS.Parents())
		if err != nil {
			return mpools, head, err
		}

	}

	return mpools, head, err
}
