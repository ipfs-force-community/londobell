package adapter

import (
	"context"
	"os"
	"time"

	"github.com/ipfs-force-community/londobell/common"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

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
	log         = logging.Logger("adapter")

	Fxlog = &fxlogger{
		ZapEventLogger: log,
	}
	Components StateComponents
)

type StateComponents struct {
	fx.In
	SM   common.StateManager
	CS   common.ChainStore
	Full v0api.FullNode
}

type fxlogger struct {
	*logging.ZapEventLogger
}

func (l *fxlogger) Printf(msg string, args ...interface{}) {
	l.ZapEventLogger.Debugf(msg, args...)
}

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
