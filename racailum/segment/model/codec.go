package model

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"

	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
)

func init() {
	addrExams := []interface{}{}
	for _, s := range []string{
		"f073366",
		"f1if5yzf6lkmpbd5jmysolhquqwekryxqdna637hq",
		"f2hhfann7xa3lay6pybsw5liunztjkkcuwptgtp5q",
		"f3unasom6mrmop7ycuunetpovwp645f4wyquqsrc5nwakg3cnxyse4ibgpcyiq3ebhitknz6zmocoi6qq6lvla",
	} {
		addrExam, err := address.NewFromString(s)
		if err != nil {
			panic(fmt.Errorf("invalid address string %s: %w", s, err))
		}

		addrExams = append(addrExams, addrExam)
	}

	addrCodecs := append([]interface{}{addressBSONEncode, addressBSONDecode}, addrExams...)

	cidExams := []interface{}{}
	for _, s := range []string{
		"bafy2bzacedxrcswo7d56zgsxqljtv7evmg7cbfnmqoxsj7ltntxkxgcaxtmkw", // blk header id
		"bafy2bzaced3ysajbgtt2gjc32hatke5fkddjpic22osxglbfukglsoh2dx744", // msg id
		"bafy2bzacecf7c2j3qvkfmiwgy3q5hbzjehwvc7t4w52zcdc3eup2m7kbj2swq", // state root
	} {
		cidExam, err := cid.Decode(s)
		if err != nil {
			panic(fmt.Errorf("invalid cid string %s: %w", s, err))
		}

		cidExams = append(cidExams, cidExam)
	}

	cidCodecs := append([]interface{}{cidBSONEncode, cidBSONDecode}, cidExams...)

	blkIDs := []cid.Cid{}
	for _, s := range []string{
		"bafy2bzacebf4c5nqq3qxvydjbx43cofztsg7q6rzq3svaincixrvjjtp4dsa2",
		"bafy2bzaceari7j4r7mwl7tyk26mhmkxsdj4hgjknenvs4gwdeeuocrqnwoodg",
		"bafy2bzaceay6rcsumesti27pjt5hhyy42vrzbvpjvbmni2foncn3guru3gysw",
	} {
		c, err := cid.Decode(s)
		if err != nil {
			panic(fmt.Errorf("invalid block header id %s: %w", s, err))
		}

		blkIDs = append(blkIDs, c)
	}

	tskExams := []interface{}{}
	for i := 1; i <= len(blkIDs); i++ {
		tskExams = append(tskExams, types.NewTipSetKey(blkIDs[:i]...))
	}

	tskCodecs := append([]interface{}{tipsetKeyBSONEncode, tipsetKeyBSONDecode}, tskExams...)

	hashExams := []interface{}{}
	for _, s := range []string{
		"0x9db82eec8a529c12361cedfaa3ae43ccc66234efac00a4bae8e0cc6dcb269be1",
		"0xc3535fefdd76f2de7a843fa4defcecb26cbc2d5b7279f7939662ca75815117eb",
		"0x9941603956a8fd47753cbbdb3be2ae35f3afd8af164ab76222b72904f9ba84b8",
	} {
		hashExam, err := ethtypes.ParseEthHash(s)
		if err != nil {
			panic(fmt.Errorf("invalid hash string %s: %w", s, err))
		}

		hashExams = append(hashExams, hashExam)
	}

	hashCodecs := append([]interface{}{ethHashBSONEncode, ethHashBSONDecode}, hashExams...)

	// encoder, decoder, examples
	codecs := [][]interface{}{
		addrCodecs,
		cidCodecs,
		tskCodecs,
		hashCodecs,
		{bigIntBSONEncode, bigIntBSONDecode, big.NewInt(1 << 10), big.NewInt(1 << 30), reward.BaselineExponent},
		{uintptrBSONEncode, uintptrBSONDecode, uintptr(1 << 10), uintptr(1 << 30)},
		{bitfieldBSONEncode, bitfieldBSONDecode, bitfield.New(), bitfield.NewFromSet([]uint64{1 << 10, 1 << 20, 1 << 30, 1 << 40})},
	}

	for i := range codecs {
		cs := codecs[i]
		if err := mcodec.RegisterCodec(cs[0], cs[1], true, cs[2:]...); err != nil {
			panic(fmt.Errorf("register #%d codec: %w", i, err))
		}
	}
}

// address.Address
func addressBSONEncode(addr address.Address) (string, bool, error) {
	if addr == address.Undef {
		return "", true, nil
	}

	return addr.String()[1:], false, nil
}

func addressBSONDecode(s string) (address.Address, error) {
	switch address.CurrentNetwork {
	case address.Mainnet:
		s = address.MainnetPrefix + s

	case address.Testnet:
		s = address.TestnetPrefix + s

	default:
		return address.Undef, fmt.Errorf("unknown current network: %v", address.CurrentNetwork)
	}

	return address.NewFromString(s)
}

// cid.Cid
func cidBSONEncode(c cid.Cid) (string, bool, error) {
	if !c.Defined() {
		return "", true, nil
	}

	return c.String(), false, nil
}

func cidBSONDecode(s string) (cid.Cid, error) {
	return cid.Decode(s)
}

// big.Int
func bigIntBSONEncode(i big.Int) (string, bool, error) {
	if i.Int == nil {
		return "", true, nil
	}

	return i.Int.String(), false, nil
}

func bigIntBSONDecode(s string) (big.Int, error) {
	return big.FromString(s)
}

// uintptr
func uintptrBSONEncode(ptr uintptr) (int64, bool, error) {
	return int64(ptr), false, nil
}

func uintptrBSONDecode(n int64) (uintptr, error) {
	return uintptr(n), nil
}

// bitfield.BitField
func bitfieldBSONEncode(bf bitfield.BitField) ([]byte, bool, error) {
	buf := bytes.NewBufferString("")
	if err := bf.MarshalCBOR(buf); err != nil {
		return nil, false, err
	}

	return buf.Bytes(), false, nil
}

func bitfieldBSONDecode(b []byte) (bitfield.BitField, error) {
	br := bytes.NewReader(b)
	var bf bitfield.BitField
	return bf, bf.UnmarshalCBOR(br)
}

// types.TipSetKey
func tipsetKeyBSONEncode(tsk types.TipSetKey) (string, bool, error) {
	if tsk == types.EmptyTSK {
		return "", true, nil
	}

	cids := tsk.Cids()
	strs := make([]string, len(cids))
	for i := range cids {
		strs[i] = cids[i].String()
	}

	return strings.Join(strs, ","), false, nil
}

func tipsetKeyBSONDecode(s string) (types.TipSetKey, error) {
	tsCids, err := lcli.ParseTipSetString(s)
	if err != nil {
		return types.EmptyTSK, fmt.Errorf("parse cids in tipset key: %w", err)
	}

	tsk := types.NewTipSetKey(tsCids...)
	return tsk, nil
}

// ethtypes.EthHash
func ethHashBSONEncode(hash ethtypes.EthHash) (string, bool, error) {
	return hash.String(), false, nil
}

func ethHashBSONDecode(s string) (ethtypes.EthHash, error) {
	return ethtypes.ParseEthHash(s)
}
