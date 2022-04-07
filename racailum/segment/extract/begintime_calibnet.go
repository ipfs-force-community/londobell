//go:build calibnet
// +build calibnet

package extract

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
)

const calibnetBeginTime = "2021-06-19T08:00:00+08:00" // 高度0时的时间

var (
	Loc, _      = time.LoadLocation("Asia/Shanghai")
	baseTime, _ = time.Parse(time.RFC3339, calibnetBeginTime)
)

func IsZeroHour(curEpoch abi.ChainEpoch) bool {
	curTime := time.Unix(baseTime.Unix()+int64(curEpoch)*30, 0).In(Loc)
	if curTime.Hour() == 0 && curTime.Minute() == 0 && curTime.Second() == 0 {
		return true
	}

	return false
}
