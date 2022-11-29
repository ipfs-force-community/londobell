package lotusCmdModel

import "github.com/ipfs/go-cid"

type ChainReadObjReq struct {
	ObjectCid string `json:"object_cid"`
}

type ChainReadObjRes struct {
	ObjectCid cid.Cid `json:"object_cid"`
	Object    []byte  `json:"object"`
}
