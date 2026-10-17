package aggregators

import (
	"errors"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
)

// TestEventsRootNotReady 锁住「事件文档查不到时是否算未就绪」的判据：
// 只有「正式库提交水位已覆盖该高度」才允许按「真的没有事件」处理，
// 否则必须让调用方按未就绪重试（不能回空日志回执，那会造成消费方静默丢数）。
func TestEventsRootNotReady(t *testing.T) {
	cases := []struct {
		name           string
		epoch          abi.ChainEpoch
		finalHeight    abi.ChainEpoch
		finalHeightErr error
		want           bool
	}{
		{
			name:        "消息高度高于正式水位：批次还没提交，事件可能还没落库 ⇒ 未就绪",
			epoch:       6464450,
			finalHeight: 6464428,
			want:        true,
		},
		{
			name:        "消息高度等于正式水位：该批次已提交，事件必已落库 ⇒ 不是未就绪",
			epoch:       6464428,
			finalHeight: 6464428,
			want:        false,
		},
		{
			name:        "消息高度低于正式水位：同上 ⇒ 不是未就绪",
			epoch:       6464400,
			finalHeight: 6464428,
			want:        false,
		},
		{
			name:           "正式水位读取失败：保守按未就绪（宁可重试，不谎报无日志）——即便高度很低",
			epoch:          6400000, //「低于水位」这个判断本身已不可信
			finalHeight:    0,
			finalHeightErr: errors.New("url not found in DBCollectionsMap"),
			want:           true,
		},
		{
			name:        "正式水位读成 0（无历史）而消息高度为正 ⇒ 未就绪",
			epoch:       1,
			finalHeight: 0,
			want:        true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := eventsRootNotReady(c.epoch, c.finalHeight, c.finalHeightErr)
			if got != c.want {
				t.Fatalf("eventsRootNotReady(epoch=%s, final=%s, err=%v) = %v, 期望 %v",
					c.epoch, c.finalHeight, c.finalHeightErr, got, c.want)
			}
		})
	}
}
