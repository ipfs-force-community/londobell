package actorstate

import (
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	"github.com/ipfs-force-community/londobell/racailum/segment/model/schema"
)

func init() {
	schema.Register(
		schema.Model{
			Name: "mining-profitability",
			D:    &model.MiningProfitability{},
		},
	)
}

var sectorSize32GiB = abi.SealProofInfos[abi.RegisteredSealProof_StackedDrg32GiBV1_1].SectorSize
