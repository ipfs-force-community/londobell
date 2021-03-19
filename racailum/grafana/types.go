package grafana

import (
	"encoding/json"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
)

var _ json.Marshaler = point{}

type pointset struct {
	Target string  `json:"target" bson:"_id"`
	Points []point `json:"datapoints" bson:"points"`
}

type point struct {
	Epoch abi.ChainEpoch
	Value float64
}

func (p point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{p.Value, epoch2time(p.Epoch).UnixNano() / int64(time.Millisecond)})
}

type searchReq struct {
	Target string `json:"target"`
}

type searchResp []string

type queryReq struct {
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`

	Targets []queryReqTarget `json:"targets"`
}

type queryReqTarget struct {
	Target string      `json:"target"`
	RefID  string      `json:"refId"`
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
}

type queryResp []pointset

type queryCtx struct {
	From int64
	To   int64
	Data interface{}
}
