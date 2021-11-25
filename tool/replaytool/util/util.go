package util

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"io/ioutil"
	"time"
)

type ReplayResult struct {
	MsgCid         cid.Cid // the message cid to find
	Msg            *types.Message
	MsgRct         *types.MessageReceipt
	ExecutionTrace types.ExecutionTrace
	Error          string
	Duration       time.Duration
}

func WriteTofile(replayResult []*api.InvocResult) error {
	var filename = "replay.txt"

	fileContent, err := json.Marshal(replayResult)
	if err = ioutil.WriteFile(filename, fileContent, 0666); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}
