//go:build !calibnet && !wallabynet
// +build !calibnet,!wallabynet

package extract

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
)

const mainnetBeginTime = "2020-08-25T06:00:00+08:00" // 高度0时的时间

var (
	Loc, _      = time.LoadLocation("Asia/Shanghai")
	BaseTime, _ = time.Parse(time.RFC3339, mainnetBeginTime)
)

func IsZeroHour(curEpoch abi.ChainEpoch) bool {
	curTime := time.Unix(BaseTime.Unix()+int64(curEpoch)*30, 0).In(Loc)
	if curTime.Hour() == 0 && curTime.Minute() == 0 && curTime.Second() == 0 {
		return true
	}

	return false
}
