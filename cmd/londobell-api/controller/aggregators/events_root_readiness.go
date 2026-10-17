package aggregators

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
)

// 事件文档（EventsRoot）就绪判据。
//
// 背景（2026-10-17 主网事故 fevm.erc_20_transfers 整高度缺行）：
// 回执里的日志来自 EventsRoot 集合（_id = 消息回执的 eventsRoot CID），由 bell 抽取管线写入，
// 与正式库提交水位（/aggregators/final_height）**同批**推进：ExtractTipSets（边抽边落库）
// → Aggregate → SaveFinalHeight，即「正式水位 ≥ H ⇒ H 的事件必已落库」。
//
// 而同步器消费用的「近链头水位」(/aggregators/latest_tipset，读 Tmp 库) 比正式水位
// 领先 17~37 个高度（8.5~18.5 分钟）。于是在这段窗口里查 EventsRoot 会**查不到**，
// 旧实现把它容忍成「空日志回执」（code:0 + logs:[]），与「这条消息真的没有日志」不可区分
// ⇒ 消费方静默丢数（实测该表约四成高度整段缺行，长期如此）。
//
// 因此这里给出判据：查不到事件文档时，是「还没落库（可重试）」还是「疑似真实缺口（需留痕）」。
//
//	消息高度 > 正式库提交水位          ⇒ 未就绪：批次还没提交，事件本来就可能还没落库；
//	消息高度 ≤ 正式库提交水位 但查不到 ⇒ 按上面的推进次序本不该发生，属真实缺口：留 ERROR 痕迹；
//	取不到正式水位（上游异常）        ⇒ 保守按未就绪处理（宁可让调用方重试，也不谎报「无日志」）。
func eventsRootNotReady(epoch abi.ChainEpoch, finalHeight abi.ChainEpoch, finalHeightErr error) bool {
	if finalHeightErr != nil {
		return true
	}
	return epoch > finalHeight
}

// formalFinalHeight 读取正式库提交水位（/aggregators/final_height 的同一实现）。
func formalFinalHeight(ctx context.Context) (abi.ChainEpoch, error) {
	h, err := GetFinalHeightForFormalDB(ctx)
	if err != nil {
		return 0, fmt.Errorf("read formal final height: %w", err)
	}
	return h, nil
}
