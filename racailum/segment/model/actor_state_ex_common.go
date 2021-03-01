package model

import (
	"fmt"
	"math"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	rlepluslazy "github.com/filecoin-project/go-bitfield/rle"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/dtynn/londobell/common"
)

// common errors
var (
	ErrRLEOverflow = fmt.Errorf("RLE+ overflows")
)

var (
	_ common.Indexed = (*ActorStateExBasic)(nil)

	actorStateExBasicEpochField = extractEpochFieldName(ActorStateExBasic{})
)

// ActorStateExBasic is the basic schema for all kinds of actor-state-exes
type ActorStateExBasic struct {
	ID    cid.Cid `bson:"_id"`
	Path  []cid.Cid
	Addr  address.Address
	Epoch abi.ChainEpoch
}

// Indexes impl common.IndexedDocument
func (a ActorStateExBasic) Indexes() [][]string {
	return [][]string{
		[]string{actorStateEpochField, "Addr"},
	}
}

// BitfieldDetail contains raw data, runs & run count
type BitfieldDetail struct {
	RawBytes int
	Runs     []rlepluslazy.Run
	RunCount int
	Count    uint64
}

// NewBitfieldDetail attempts to get details from a bitfield.BitField
// for Count logic, see https://github.com/filecoin-project/go-bitfield/blob/v0.2.4/rle/runs.go#L123-L143
func NewBitfieldDetail(bf bitfield.BitField, withRuns bool) (BitfieldDetail, error) {
	raw, _, err := bitfieldBSONEncode(bf)
	if err != nil {
		return BitfieldDetail{}, err
	}

	iter, err := bf.RunIterator()
	if err != nil {
		return BitfieldDetail{}, err
	}

	var length uint64
	var count uint64

	runs := make([]rlepluslazy.Run, 0, 64)
	for iter.HasNext() {
		next, err := iter.NextRun()
		if err != nil {
			return BitfieldDetail{}, err
		}

		if math.MaxUint64-next.Len < length {
			return BitfieldDetail{}, ErrRLEOverflow
		}

		length += next.Len

		if next.Val {
			count += next.Len
		}

		runs = append(runs, next)
	}

	runCount := len(runs)
	if !withRuns {
		runs = nil
	}

	return BitfieldDetail{
		RawBytes: len(raw),
		Runs:     runs,
		RunCount: runCount,
		Count:    count,
	}, nil
}
