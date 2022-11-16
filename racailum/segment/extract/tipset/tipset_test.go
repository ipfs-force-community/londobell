package tipset

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ipfs-force-community/londobell/racailum/segment/model"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/crypto/sha3"
)

func TestHexEncodeParams(t *testing.T) {
	entryPoint, err := hex.DecodeString("a9059cbb")
	if err != nil {
		return
	}
	var inputData []byte
	inputData, err = hex.DecodeString("000000000000000000000000ff00000000000000000000000000000000000064")
	if err != nil {
		return
	}
	// TODO need to encode as CBOR bytes now
	params := append(entryPoint, inputData...)

	var buffer bytes.Buffer
	if err := cbg.WriteByteArray(&buffer, params); err != nil {
		return
	}
	params = buffer.Bytes()

	// encode
	enbuffer := bytes.NewBuffer(params)
	enParams, err := cbg.ReadByteArray(enbuffer, 1024)
	if err != nil {
		return
	}
	fmt.Println(hex.EncodeToString(enParams))
}

func TestGetStringsIndex(t *testing.T) {
	s := "withdraw(address token, uint256 amount, address destination)"
	a := strings.Index(s, "(")
	b := strings.Index(s, ")")
	fmt.Println(a, b)

	type params struct {
		Type string
		Name string
	}

	s1 := s[9:59]
	fmt.Println("s1:", s1)
	s2 := strings.Split(s1, ",")
	fmt.Println("s2:", s2)
	m := make([]params, 0)
	for i := range s2 {
		s := s2[i]
		s = strings.TrimSpace(s)
		s3 := strings.Split(s, " ")
		fmt.Println("s3:", s3)
		m = append(m, params{s3[0], s3[1]})
	}

	fmt.Println(m)
}

func TestKeccak256Hash(t *testing.T) {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte("balanceOf(address)"))
	hash := hexutil.Encode(hasher.Sum(nil)[:])
	fmt.Println(hash)
}

const (
	balanceOf   = "balanceOf(address account)"
	totalSupply = "totalSupply()"
	withdraw    = "withdraw(address token, uint256 amount, address destination)"
)

func TestSearchConstractMethod(t *testing.T) {
	functionList := []string{balanceOf, totalSupply, withdraw}

	if err := model.RegistryConstractMethods(functionList); err != nil {
		panic(err)
	}

	fmt.Println(model.ConstractMethods())

	fmt.Println(model.SearchConstractMethod(fmt.Sprintf("%s%s", "0x", "70a08231")))
}
