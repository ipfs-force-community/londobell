package lotusCmdModel

import (
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/ipfs/go-cid"
)

type StateActorCIDsReq struct {
	NetworkVersion uint `json:"network_version"`
}

type StateActorCIDsRes struct {
	NetworkVersion network.Version     `json:"network_version"`
	ActorVersion   actorstypes.Version `json:"actor_version"`
	ManifestCid    cid.Cid             `json:"manifest_cid"`
	ActorsCids     map[string]cid.Cid  `json:"actors_cids"`
}
