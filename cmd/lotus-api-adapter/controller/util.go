package controller

import (
	"context"
	"os"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
)

const (
	mainnetBeginTime = "2020-08-25T06:00:00+08:00" // 高度0时的时间
)

var (
	Loc, _      = time.LoadLocation("Asia/Shanghai")
	baseTime, _ = time.Parse(time.RFC3339, mainnetBeginTime)
	API         v0api.FullNode
	log         = logging.Logger("racailum")
)

func CalcTimeByEpoch(height uint64) time.Time {
	return time.Unix(baseTime.Unix()+int64(height)*30, 0).In(Loc)
}

func GetFullNodeAPI(ctx context.Context) (err error) {
	url := os.Getenv("LOTUS_URL")
	API, _, err = client.NewFullNodeRPCV0(ctx, url, nil)
	if err != nil {
		return err
	}

	return nil
}
