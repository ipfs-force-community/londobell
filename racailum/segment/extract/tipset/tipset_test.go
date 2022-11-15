package tipset

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	cbg "github.com/whyrusleeping/cbor-gen"
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
