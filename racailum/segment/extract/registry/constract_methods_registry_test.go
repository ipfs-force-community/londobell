package registry

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/stretchr/testify/require"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/crypto/sha3"
)

func TestHexEncodeParams(t *testing.T) {
	var (
		point = "a9059cbb"
		input = "000000000000000000000000ff00000000000000000000000000000000000064"
	)

	entryPoint, err := hex.DecodeString(point)
	require.Equal(t, nil, err)

	var inputData []byte
	inputData, err = hex.DecodeString(input)
	require.Equal(t, nil, err)

	// TODO need to encode as CBOR bytes now
	params := append(entryPoint, inputData...)

	var buffer bytes.Buffer
	err = cbg.WriteByteArray(&buffer, params)
	require.Equal(t, nil, err)

	// encode
	params = buffer.Bytes()
	hexParams, err := HexEncodeByteArray(params)
	require.Equal(t, nil, err)
	require.Equal(t, fmt.Sprintf("%s%s", point, input), *(*string)(unsafe.Pointer(&hexParams)))
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
	_, err := hasher.Write([]byte("balanceOf(address)"))
	require.Equal(t, nil, err)
	hash := hexutil.Encode(hasher.Sum(nil)[:])

	require.Equal(t, true, strings.HasPrefix(hash, "0x"))
	require.Equal(t, 64, len(hash[2:]))
	require.Equal(t, balanceOfID, hash[:10])
}

const (
	balanceOfID   = "0x70a08231"
	totalSupplyID = "0x18160ddd"
	withdrawID    = "0x69328dec"

	invalid1   = "invalid1(,)"
	invalid2   = "invalid2(, ,)"
	test       = "test( )"
	balanceOf2 = "balanceOf(address)"
)

var (
	functionList        = []string{balanceOf, totalSupply, withdraw}
	methodIDList        = []string{balanceOfID, totalSupplyID, withdrawID}
	invalidFunctionList = []string{invalid1, invalid2, test}
)

func TestGetMethodID(t *testing.T) {
	methodID, err := GetMethodID(balanceOf)
	require.Equal(t, nil, err)
	require.Equal(t, balanceOfID, methodID)

	methodID2, err := GetMethodID(balanceOf2)
	require.Equal(t, nil, err)
	require.Equal(t, balanceOfID, methodID2)
}

func TestGetConstractParams(t *testing.T) {
	for _, function := range functionList {
		constractParams, err := GetConstractParams(function)
		require.Equal(t, nil, err)

		if function == balanceOf {
			require.Equal(t, 1, len(constractParams))
		}
		if function == totalSupply {
			require.Equal(t, 0, len(constractParams))
		}
		if function == withdraw {
			require.Equal(t, 3, len(constractParams))
		}

		for i, param := range constractParams {
			if function == balanceOf {
				require.Equal(t, "account", param.Name)
				require.Equal(t, "address", param.Type)
				require.Equal(t, "", param.Data)
			}

			if function == withdraw {
				if i == 0 {
					require.Equal(t, "token", param.Name)
					require.Equal(t, "address", param.Type)
					require.Equal(t, "", param.Data)
				}
				if i == 1 {
					require.Equal(t, "amount", param.Name)
					require.Equal(t, "uint256", param.Type)
					require.Equal(t, "", param.Data)
				}
				if i == 2 {
					require.Equal(t, "destination", param.Name)
					require.Equal(t, "address", param.Type)
					require.Equal(t, "", param.Data)
				}
			}
		}

	}

	for _, invalidFunction := range invalidFunctionList {
		constractParams, err := GetConstractParams(invalidFunction)
		require.Equal(t, nil, err)
		require.Equal(t, 0, len(constractParams))
	}

}

func TestRegistryConstractMethods(t *testing.T) {
	if err := RegisterConstractMethods(functionList); err != nil {
		panic(err)
	}

	methods := ConstractMethods()
	require.Equal(t, 3, len(methods))

	// registry successfully
	for _, methodID := range methodIDList {
		_, ok := methods[methodID]
		require.Equal(t, true, ok)
	}

	// methodID -> functionName
	for methodID, inputData := range methods {
		if methodID == balanceOfID {
			require.Equal(t, balanceOf, inputData.Function)
		}
		if methodID == totalSupplyID {
			require.Equal(t, totalSupply, inputData.Function)
		}
		if methodID == withdrawID {
			require.Equal(t, withdraw, inputData.Function)
		}
	}

	// methodID -> ConstractParams
	for methodID, inputData := range methods {
		params := inputData.Params

		if methodID == balanceOfID {
			require.Equal(t, 1, len(params))
			require.Equal(t, "account", params[0].Name)
			require.Equal(t, "address", params[0].Type)
		}
		if methodID == totalSupplyID {
			require.Equal(t, 0, len(params))
		}
		if methodID == withdrawID {
			require.Equal(t, 3, len(params))
			for i := range params {
				if i == 0 {
					require.Equal(t, "token", params[i].Name)
					require.Equal(t, "address", params[i].Type)
				}
				if i == 1 {
					require.Equal(t, "amount", params[i].Name)
					require.Equal(t, "uint256", params[i].Type)
				}
				if i == 2 {
					require.Equal(t, "destination", params[i].Name)
					require.Equal(t, "address", params[i].Type)
				}
			}
		}
	}

}

func TestGetType(t *testing.T) {
	p := HexString("hello")
	require.Equal(t, true, reflect.TypeOf(cbor.Er(p)) == reflect.ValueOf(HexString("")).Type())
}

func TestAssignData(t *testing.T) {
	datas := "000000000000000000000000ff00000000000000000000000000000000000064"
	p, ok, err := AssignDataForConstractParams(balanceOfID, datas)
	require.Equal(t, true, ok)
	require.NoError(t, err, fmt.Errorf("falied"))
	require.Equal(t, p.Function, balanceOf)
	require.Equal(t, 1, len(p.Params))
	require.Equal(t, "account", p.Params[0].Name)
	require.Equal(t, "address", p.Params[0].Type)
	require.Equal(t, fmt.Sprintf("%s%s", "0x", datas), p.Params[0].Data)
}
