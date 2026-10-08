package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

// rewardStreamDenom 是奖励流评估权重与受益方份额的定点分母（go-state-types v19
// builtin/v19/reward.Denom = 1e18）。契约 §8.1 要求原样返回，故固定成字符串常量。
const rewardStreamDenom = "1000000000000000000"

// GetRewardStreamLedger 读 f02 奖励 actor 的流账本（NV29/FIP-0118），暴露给
// filscan_backend。契约 D：.hermes/plans/2026-10-08-nv29-adapt/README.md §8.1。
//
// 账本一律走 lotus 的 reward.State.StreamLedger（lotus v1.37.0
// chain/actors/builtin/reward/reward.go:204，实现 v19.go:109-176），本仓不自己解
// StreamsRoot 的 CBOR。请求体 {"epoch":N}；epoch 为 0 或缺省时取链头。
func GetRewardStreamLedger(c *gin.Context) {
	alog := log.With("method", "GetRewardStreamLedger")
	req := model.EpochReq{}
	res := model.CommonRes{Code: model.Success}

	// epoch 可省略：空 body / {} / {"epoch":0} 一律取链头。
	if err := c.BindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := fullnode.API.GetAppropriateAPI()

	var (
		ts  *types.TipSet
		err error
	)
	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

	ract, err := api.StateGetActor(ctx, reward.Address, ts.Key())
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	rst, err := reward.Load(stor, ract)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData, err := buildRewardStreamLedger(ts.Height(), rst)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}

// buildRewardStreamLedger 把 f02 在给定高度的账本映射成契约 §8.1 的响应。
//
// v18（NV29 未激活）不调用 StreamLedger——它对 v18 返回
// "reward streams are unsupported in actors v18"（lotus v19 之前的 state*.go 皆然）：
// 直接给 nv29=false + 空 streams/tombstones，不报错（主网当前即此分支）。
// v19 读取失败（StreamsRoot 缺失/解码错、应计行与显式流不配对等）返回明确错误，
// 绝不静默降级成空数据。
func buildRewardStreamLedger(epoch abi.ChainEpoch, st reward.State) (model.RewardStreamLedgerRes, error) {
	out := model.RewardStreamLedgerRes{
		Epoch:      int64(epoch),
		NV29:       st.ActorVersion() >= actorstypes.Version19,
		Denom:      rewardStreamDenom,
		Streams:    []model.RewardLedgerStream{},
		Tombstones: []model.RewardLedgerTombstone{},
		Liability:  "0",
	}
	if !out.NV29 {
		return out, nil
	}

	ledger, err := st.StreamLedger(epoch)
	if err != nil {
		return out, fmt.Errorf("read reward stream ledger at epoch %d: %w", epoch, err)
	}

	for _, stream := range ledger.Streams {
		out.Streams = append(out.Streams, rewardLedgerStream(stream))
	}
	for _, tombstone := range ledger.Tombstones {
		out.Tombstones = append(out.Tombstones, rewardLedgerTombstone(tombstone))
	}
	out.Liability = ledger.Liability().String()

	return out, nil
}

func rewardLedgerStream(s reward.Stream) model.RewardLedgerStream {
	return model.RewardLedgerStream{
		ID:              int64(s.ID),
		Implicit:        s.Implicit,
		EvaluatedWeight: strconv.FormatUint(s.EvaluatedWeight, 10),
		Accrued:         s.Accrued.String(),
		ClaimedPeriod:   sumRecipientAmounts(s.ClaimedPeriod).String(),
		Payable:         sumRecipientAmounts(s.Payable).String(),
		Recipients:      rewardLedgerRecipients(s),
	}
}

// rewardLedgerRecipients 合并份额表与两个余额表：先按份额表（链上按受益人排序）输出，
// 再把只出现在 Payable/ClaimedPeriod 里的遗留受益方（换址等）追加在后，保证余额不丢。
func rewardLedgerRecipients(s reward.Stream) []model.RewardLedgerRecipient {
	payable := amountsByRecipient(s.Payable)
	claimed := amountsByRecipient(s.ClaimedPeriod)

	out := make([]model.RewardLedgerRecipient, 0, len(s.Shares))
	seen := make(map[string]struct{}, len(s.Shares))

	appendRecipient := func(addr string, share uint64) {
		seen[addr] = struct{}{}
		out = append(out, model.RewardLedgerRecipient{
			Address:       addr,
			Share:         strconv.FormatUint(share, 10),
			Payable:       recipientAmount(payable, addr).String(),
			ClaimedPeriod: recipientAmount(claimed, addr).String(),
		})
	}

	for _, share := range s.Shares {
		appendRecipient(share.Recipient.String(), share.Share)
	}
	for _, rows := range [][]reward.RecipientAmount{s.Payable, s.ClaimedPeriod} {
		for _, row := range rows {
			if addr := row.Recipient.String(); !hasRecipient(seen, addr) {
				appendRecipient(addr, 0)
			}
		}
	}

	return out
}

func rewardLedgerTombstone(t reward.Tombstone) model.RewardLedgerTombstone {
	out := model.RewardLedgerTombstone{
		ID:         int64(t.ID),
		Recipients: make([]model.RewardLedgerTombRecipient, 0, len(t.Payable)),
	}
	for _, row := range t.Payable {
		out.Recipients = append(out.Recipients, model.RewardLedgerTombRecipient{
			Address: row.Recipient.String(),
			Payable: row.Amount.String(),
		})
	}
	return out
}

func amountsByRecipient(rows []reward.RecipientAmount) map[string]abi.TokenAmount {
	out := make(map[string]abi.TokenAmount, len(rows))
	for _, row := range rows {
		out[row.Recipient.String()] = row.Amount
	}
	return out
}

// recipientAmount 取受益方余额；缺失或零值时返回 big.Zero()，避免 big.Int 的零值
// （内嵌 *big.Int 为 nil）在 String() 上 panic。
func recipientAmount(m map[string]abi.TokenAmount, addr string) abi.TokenAmount {
	if v, ok := m[addr]; ok && v.Int != nil {
		return v
	}
	return big.Zero()
}

func sumRecipientAmounts(rows []reward.RecipientAmount) abi.TokenAmount {
	total := big.Zero()
	for _, row := range rows {
		total = big.Add(total, row.Amount)
	}
	return total
}

func hasRecipient(set map[string]struct{}, addr string) bool {
	_, ok := set[addr]
	return ok
}
