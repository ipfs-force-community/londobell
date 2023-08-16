package model

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/verifreg"

	"github.com/ipfs-force-community/londobell/common"
)

var (
	_                      common.IndexedDocument = (*ChangedClaim)(nil)
	changedClaimEpochField                        = extractEpochFieldName(ChangedClaim{})
	changedClaimColName                           = getColName(ChangedClaim{})
)

type ChangedClaim struct {
	ID             string `bson:"_id"`
	ClaimID        verifreg.ClaimId
	Epoch          abi.ChainEpoch
	verifreg.Claim `bson:",inline"`
	Added          bool // new created
	Removed        bool
}

func NewChangedClaim(claimID verifreg.ClaimId, claim verifreg.Claim, epoch abi.ChainEpoch, added, removed bool) *ChangedClaim {
	return &ChangedClaim{
		ID:      fmt.Sprintf("%v-%v", claimID, epoch),
		ClaimID: claimID,
		Epoch:   epoch,
		Claim:   claim,
		Added:   added,
		Removed: removed,
	}
}

// CollectionName impl CollectionName
func (s *ChangedClaim) CollectionName() string {
	return changedClaimColName
}

// EpochField impl common.Document
func (s *ChangedClaim) EpochField() *string {
	return &changedClaimEpochField
}

// ResetPolicy impl common.Document
func (s *ChangedClaim) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(changedClaimEpochField, lower, upper), true
}

func (s *ChangedClaim) Indexes() [][]string {
	return [][]string{
		[]string{"Provider", changedClaimEpochField, "ClaimID", "Added"},
		[]string{"Provider", changedClaimEpochField, "ClaimID", "Removed"},
		[]string{"Provider", "Sector"},
		[]string{"ClaimID", changedClaimEpochField},
	}
}

func (s *ChangedClaim) IsMutable() bool {
	return false
}
