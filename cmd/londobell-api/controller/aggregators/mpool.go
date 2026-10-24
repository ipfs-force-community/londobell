package aggregators

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/adapter"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/controller/aggregators/common"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
	"github.com/ipfs/go-cid"
)

var (
	poolCache = make(map[cid.Cid]int64)
	mu        sync.Mutex
)

func RefreshMpool() {
	ticker := time.NewTicker(time.Second)
	tlog := log.With("task", "RefreshMpool")
	for range ticker.C {

		api := fullnode.API.GetAppropriateAPI()
		unStoredMsgs, _, err := getUnStoredMsgs(context.Background(), api)
		if err != nil {
			tlog.Error(err)
			break
		}
		refreshMpoolCache(unStoredMsgs)

	}
}

func refreshMpoolCache(unStoredMsgs map[*types.TipSet][]*types.SignedMessage) {
	// 并发契约：本函数被 1s ticker（RefreshMpool）与 HTTP handler（GetMpool）同时调用，
	// 而 Go 的 map 并发读写是 fatal error（直接 exit 2，不是可恢复的 panic）。
	// 所以对包级 poolCache 只做「持锁取快照 → 遍历快照写本地 newCache → 持锁换指针」，
	// 遍历期间一个字节都不碰全局变量。
	mu.Lock()
	prev := poolCache
	mu.Unlock()

	newCache := make(map[cid.Cid]int64)
	for _, msgs := range unStoredMsgs {
		for _, msg := range msgs {
			ts, ok := prev[msg.Cid()]
			if !ok {
				ts = time.Now().Unix()
			}
			newCache[msg.Cid()] = ts
		}
	}

	mu.Lock()
	poolCache = newCache
	mu.Unlock()
}

// poolCacheAt 读单个 CID 的首次出现时间。handler 读侧与上面的刷新写侧并发，
// 所以读也必须持锁（曾经直接在 handler 里读 poolCache[msg.Cid()]，与 ticker 打架）。
func poolCacheAt(c cid.Cid) int64 {
	mu.Lock()
	defer mu.Unlock()
	return poolCache[c]
}

func GetMpool(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
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

	// poolItem 是一条待打包消息连同它所属的 tipset；hash 分支与默认分支都用它。
	type poolItem struct {
		cur *types.TipSet
		msg *types.SignedMessage
	}

	unStoredMsgs, head, err := getUnStoredMsgs(ctx, api)
	refreshMpoolCache(unStoredMsgs)

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

					hash, err := util.NewEthHashFromSignedMessage(msg)
					if err != nil {
						alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
						util.ReturnOnErr(c, err)
						return
					}

					res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
						Cid: msg.Message.Cid(), SignedCid: msg.Cid(), Epoch: cur.Height(),
						From: msg.Message.From, To: msg.Message.To, Value: msg.Message.Value,
						GasLimit: msg.Message.GasLimit, GasPremium: msg.Message.GasPremium,
						Method: methodName, Hash: hash.String(), MsgTime: poolCacheAt(msg.Cid()),
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
			alog.Warnf("mcid: %s dont exist in db nor unStoredMsgs,epoch : %d", mcid, head.Height())
			msg, err := api.ChainGetMessage(ctx, mcid)
			if err != nil {
				alog.Error(fmt.Errorf("ChainGetMessage failed: %v, smsg: %v", err, mcid))
				util.ReturnOnErr(c, err)
				return
			}
			var smsg = new(types.SignedMessage)
			if msg != nil {
				smsg.Message = *msg
				methodName, err := adapter.GetMethodName(ctx, alog, api, smsg, head)
				if err != nil && err != util.ErrNotFound {
					alog.Error(err)
					util.ReturnOnErr(c, err)
					return
				}
				hash, err := util.NewEthHashFromSignedMessage(smsg)
				if err != nil {
					alog.Error(fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, msg.Cid()))
					util.ReturnOnErr(c, err)
					return
				}
				// 当mcid与smsg.Message.Cid()不同时说明消息被覆盖掉了
				if mcid == smsg.Message.Cid() {
					res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{{
						Cid: smsg.Message.Cid(), SignedCid: msg.Cid(), Epoch: head.Height(),
						From: smsg.Message.From, To: smsg.Message.To, Value: smsg.Message.Value,
						GasLimit: smsg.Message.GasLimit, GasPremium: smsg.Message.GasPremium,
						Method: methodName, Hash: hash.String(), MsgTime: poolCacheAt(msg.Cid()),
					}}}
					c.JSON(http.StatusOK, res)
					return
				}

			} else {
				log.Warnf("mcid: %s not exist in chain", mcid)
			}

		}

		c.JSON(http.StatusOK, res)
		return
	}

	if req.Hash != "" {
		// 原实现把 g.Wait() 和「返回响应」写在了 tipset 循环体内，所以只搜第一个 tipset 就返回。
		// unStoredMsgs 是 map，遍历顺序随机 ⇒ 同一个 eth hash 前后两次请求结果会不一样
		// （2026-10-24 实测：池内 76 条里 46 条查不回来）。
		// 这里改成：扫完所有 tipset 再返回；并发仍然固定（util.RunBounded），不是每条一个 goroutine。
		hashItems := make([]poolItem, 0, len(unStoredMsgs))
		for cur, msgs := range unStoredMsgs {
			for _, msg := range msgs {
				hashItems = append(hashItems, poolItem{cur: cur, msg: msg})
			}
		}

		var (
			matchMu sync.Mutex
			matched *model.PendingMessage
		)
		if err := util.RunBounded(ctx, len(hashItems), func(ctx context.Context, idx int) error {
			it := hashItems[idx]

			hash, err := util.NewEthHashFromSignedMessage(it.msg)
			if err != nil {
				return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, it.msg.Cid())
			}
			if req.Hash != hash.String() {
				return nil
			}

			methodName, err := adapter.GetMethodName(ctx, alog, api, it.msg, it.cur)
			if err != nil && err != util.ErrNotFound {
				return err
			}

			matchMu.Lock()
			if matched == nil { // eth hash 唯一，取第一条即可
				matched = &model.PendingMessage{
					Cid: it.msg.Message.Cid(), SignedCid: it.msg.Cid(), Epoch: it.cur.Height(),
					From: it.msg.Message.From, To: it.msg.Message.To, Value: it.msg.Message.Value,
					GasLimit: it.msg.Message.GasLimit, GasPremium: it.msg.Message.GasPremium,
					Method: methodName, Hash: hash.String(), MsgTime: poolCacheAt(it.msg.Cid()),
				}
			}
			matchMu.Unlock()

			return nil
		}); err != nil {
			alog.Error(err)
			util.ReturnOnErr(c, err)
			return
		}

		if matched != nil {
			res.Data = model.PendingMessagesRes{TotalCount: 1, PendingMessages: []model.PendingMessage{*matched}}
		}

		// 无论查没查到都必须在 hash 分支返回：原实现在池子为空时会落到下面的通用分页分支，
		// 把整个 pending 列表当成「这个 hash 的结果」返回。
		c.JSON(http.StatusOK, res)
		return
	}

	items := make([]poolItem, 0, len(unStoredMsgs))
	for cur, msgs := range unStoredMsgs {
		for _, msg := range msgs {
			items = append(items, poolItem{cur: cur, msg: msg})
		}
	}

	// 每项最多产出一条，按下标写回后再汇总：顺序与原来的「排序前」不一致也没关系，
	// 下面按 MsgTime 重排，结果集与总量不变。
	built := make([][]model.PendingMessage, len(items))
	if err := util.RunBounded(ctx, len(items), func(ctx context.Context, idx int) error {
		it := items[idx]

		methodName, err := adapter.GetMethodName(ctx, alog, api, it.msg, it.cur)
		if err != nil && err != util.ErrNotFound {
			return err
		}

		// todo: InvokeEVM和其他未定义方法区分开??
		// filter by methodName
		if req.MethodName != "" && methodName != req.MethodName {
			return nil
		}

		hash, err := util.NewEthHashFromSignedMessage(it.msg)
		if err != nil {
			return fmt.Errorf("newEthTxFromSignedMessage failed: %v, smsg: %v", err, it.msg.Cid())
		}

		// todo: methodName为空时加入mpool吗？
		built[idx] = []model.PendingMessage{{
			Cid: it.msg.Message.Cid(), SignedCid: it.msg.Cid(), Epoch: it.cur.Height(),
			From: it.msg.Message.From, To: it.msg.Message.To, Value: it.msg.Message.Value,
			GasLimit: it.msg.Message.GasLimit, GasPremium: it.msg.Message.GasPremium,
			Method: methodName, Hash: hash.String(), MsgTime: poolCacheAt(it.msg.Cid()),
		}}

		return nil
	}); err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	pendingMessages := make([]model.PendingMessage, 0, len(items))
	for _, b := range built {
		pendingMessages = append(pendingMessages, b...)
	}
	totalCount := int64(len(pendingMessages))

	// sort
	sort.Slice(pendingMessages, func(i, j int) bool {
		return pendingMessages[i].MsgTime > pendingMessages[j].MsgTime
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
