package util

import (
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"strings"
)

func ParseStringToCidArray(source string) ([]cid.Cid, error) {
	strs := strings.Split(source, ",")

	var cids []cid.Cid
	for _, s := range strs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	return cids, nil
}

func ParseTipSetKey(s string) (types.TipSetKey, error) {
	cids, err := ParseStringToCidArray(s)
	if err != nil {
		return types.EmptyTSK, err
	}

	return types.NewTipSetKey(cids...), nil
}
