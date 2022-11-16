package model

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/filecoin-project/go-state-types/abi"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/sha3"
)

func rangedFilter(field string, lower, upper *abi.ChainEpoch) bson.M {
	inner := bson.M{}
	if lower != nil {
		inner["$gt"] = *lower
	}

	if upper != nil {
		inner["$lt"] = *upper
	}

	return bson.M{field: inner}
}

func getColName(doc interface{}) string {
	return reflect.TypeOf(doc).Name()
}

func extractEpochFieldName(doc interface{}) string {
	field, ok := reflect.TypeOf(doc).FieldByName("Epoch")
	if !ok {
		panic(fmt.Errorf("Epoch field not found for %T", doc))
	}

	if tagstr := field.Tag.Get("bson"); tagstr != "" {
		if name := strings.Split(tagstr, ",")[0]; name != "" {
			return name
		}
	}

	return field.Name
}

type ConstractMethodsRegistry struct {
	sync.RWMutex
	c map[string]InputData
}

type InputData struct {
	Function string
	Params   []ConstractParams
}

type ConstractParams struct {
	Name string
	Type string
	Data string
}

var MethodsRegistry = struct {
	methods *ConstractMethodsRegistry
}{
	methods: &ConstractMethodsRegistry{
		c: make(map[string]InputData),
	},
}

//// RegistryMethodID: method(params...)
//balanceOf(address):70a08231
//totalSupply(): 18160ddd
//transfer(address, uint256): 9d61d234
//load(): 86d5c4be
//withdraw(address,uint256,address): 69328dec

// get from db
const (
	balanceOf   = "balanceOf(address account)"
	totalSupply = "totalSupply()"
	withdraw    = "withdraw(address token, uint256 amount, address destination)"
)

// known
// todo: 返回类型  json abi
func init() {
	// get from db
	functionList := []string{balanceOf, totalSupply, withdraw}

	if err := RegistryConstractMethods(functionList); err != nil {
		panic(err)
	}
}

func RegistryConstractMethods(functions []string) error {
	var (
		methodID  string
		inputData InputData
		err       error
	)
	MethodsRegistry.methods.Lock()

	// lock内计算，会导致lock锁占有过长时间？？
	for _, function := range functions {
		methodID, err = getMethodID(function)
		if err != nil {
			return err
		}

		params, err := getConstractParams(function)
		if err != nil {
			return err
		}

		inputData.Function = function
		inputData.Params = params
		MethodsRegistry.methods.c[methodID] = inputData
	}

	MethodsRegistry.methods.Unlock()

	// todo: 落库

	return nil
}

func getMethodID(function string) (string, error) {
	constractParams, err := getConstractParams(function)
	if err != nil {
		return "", err
	}

	var paramsList []byte
	for i, params := range constractParams {
		paramsList = append(paramsList, params.Type...)
		if i < len(constractParams)-1 {
			paramsList = append(paramsList, ',')
		}
	}

	var all []byte
	all = append(all, getFunctionName(function)...)
	all = append(all, '(')
	all = append(all, paramsList...)
	all = append(all, ')')

	var buffer bytes.Buffer
	_, err = buffer.Write(all)
	if err != nil {
		return "", err
	}

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(buffer.Bytes())
	hash := hexutil.Encode(hasher.Sum(nil)[:])

	if !strings.HasPrefix(hash, "0x") {
		return "", fmt.Errorf("invalid hex hash: %v", hash)
	}
	if len(hash) < 66 {
		return "", fmt.Errorf("too short length: %v", len(hash))
	}

	return hash[:10], nil
}

func getConstractParams(function string) ([]ConstractParams, error) {
	start := strings.Index(function, "(")
	end := strings.Index(function, ")")

	if start >= end {
		return nil, fmt.Errorf("invalid function, start %v >= end %v", start, end)
	}

	params := strings.Split(function[start+1:end], ",")
	constractParams := make([]ConstractParams, 0)
	for i := range params {
		param := params[i]
		param = strings.TrimSpace(param)
		tuples := strings.Split(param, " ")
		if len(tuples) == 2 {
			constractParams = append(constractParams, ConstractParams{Type: tuples[0], Name: tuples[1]})
		} else if len(tuples) == 1 {
			constractParams = append(constractParams, ConstractParams{Type: tuples[0]})
		} else {
			return nil, fmt.Errorf("invalid params: %v", tuples)
		}
	}

	return constractParams, nil
}

func getFunctionName(function string) string {
	start := strings.Index(function, "(")
	return function[:start]
}

func ConstractMethods() map[string]InputData {
	MethodsRegistry.methods.RLock()
	defer MethodsRegistry.methods.RUnlock()

	return MethodsRegistry.methods.c
}

func SearchConstractMethod(methodID string) (InputData, bool) {
	MethodsRegistry.methods.RLock()
	inputData, ok := MethodsRegistry.methods.c[methodID]
	MethodsRegistry.methods.RUnlock()

	return inputData, ok
}

func hexEncodeByteArray(params []byte) (string, error) {
	buffer := bytes.NewBuffer(params)
	hexParams, err := cbg.ReadByteArray(buffer, 1024)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hexParams), nil
}
