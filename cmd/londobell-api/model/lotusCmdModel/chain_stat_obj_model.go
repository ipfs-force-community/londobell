package lotusCmdModel

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/ipfs/go-cid"
)

type ChainStatObjReq struct {
	Cid  string `json:"cid"`
	Base string `json:"base"`
}

type ChainStatObjRes struct {
	Cid     cid.Cid     `json:"cid"`
	Base    cid.Cid     `json:"base"`
	ObjStat api.ObjStat `json:"obj_stat"`
}
