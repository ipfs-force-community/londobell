package util

import (
	"fmt"
	"sync"

	logging "github.com/ipfs/go-log/v2"

	"github.com/robertkrimen/otto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var log = logging.Logger("parse")

// Object classes
//
//nolint:deadcode
const (
	Object   = "Object"
	Function = "Function"
	Array    = "Array"
	String   = "String"
	Number   = "Number"
	Boolean  = "Boolean"
	Date     = "Date"
	RegExp   = "RegExp"
)

// filfox: AddLockedFund
// todo: update upgrade
// Other is ""
var AllMethodList = []string{
	"AddBalance", "AddVerifiedClient", "AllowanceExported", "Approve", "AuthenticateMessage",
	"Cancel", "CancelExported", "ChangeBeneficiary", "ChangeMultiaddrs", "ChangeOwnerAddress", "ChangePeerID",
	"ChangeWorkerAddress", "Collect", "CompactPartitions", "CompactSectorNumbers", "ConfirmChangeWorkerAddress",
	"ConfirmUpdateWorkerKey", "Constructor", "ControlAddresses", "CreateExternal", "CreateMiner",
	"DeclareFaults", "DeclareFaultsRecovered", "DisputeWindowedPoSt",
	"Exec", "ExtendClaimTerms", "ExtendSectorExpiration", "ExtendSectorExpiration2",
	"GenerateSectorLocation", "GenerateSectorLocationExported",
	"GetBeneficiary", "GetNominalSectorExpiration", "GetNominalSectorExpirationExported",
	"IncreaseAllowanceExported", "InvokeContract",
	"Other",
	"PreCommitSector", "PreCommitSectorBatch", "PreCommitSectorBatch2", "Propose", "ProveCommitAggregate",
	"ProveCommitSector", "ProveCommitSectors3", "ProveCommitSectorsNI", "ProveReplicaUpdates", "ProveReplicaUpdates2",
	"ProveReplicaUpdates3", "PubkeyAddress", "PublishStorageDeals",
	"RemoveExpiredAllocations", "RemoveSigner", "RepayDebt", "ReportConsensusFault",
	"Send", "Send(ethaccount)", "Send(placeholder)", "Settle", "SubmitWindowedPoSt", "SwapSigner",
	"TerminateSectors", "TransferFromExported",
	"UpdateChannelState",
	"ValidateSectorStatus", "ValidateSectorStatusExported",
	"WithdrawBalance",
}

var (
	// scriptCache 缓存编译好的 pipeline 脚本：otto 的 *Script 只依赖 otto 运行时版本、
	// 不绑定 VM 实例（见 otto/script.go 的注释），因此可以跨 VM 复用，省掉每次
	// 解析 + 编译 pipeline 源码的开销。
	scriptCache sync.Map // map[string]*otto.Script

	// vmPool 复用 JS 解释器：otto.New() 会构造整套 ECMAScript 全局环境，原来
	// 「每个请求 × 每个库」都新建一份，是聚合器内存与 CPU 的主要开销之一。
	vmPool = sync.Pool{New: func() interface{} { return otto.New() }}
)

// Parse generates a aggregation pipeline from the given source code with context
func Parse(ctx, src interface{}) (interface{}, error) {
	vm := vmPool.Get().(*otto.Otto)
	defer func() {
		// 归池前清掉 ctx：池化 VM 的全局作用域是共享的，不清会把上一个请求的参数
		// 泄漏给下一个请求（尤其下一个请求 ctx 为 nil 时），表现为静默查错。
		_ = vm.Set("ctx", otto.UndefinedValue())
		vmPool.Put(vm)
	}()

	script, err := compiledScript(vm, src)
	if err != nil {
		return nil, fmt.Errorf("eval source: %w", err)
	}

	if ctx != nil {
		if err := vm.Set("ctx", ctx); err != nil {
			return nil, fmt.Errorf("set context: %w", err)
		}
	}

	v, err := vm.Run(script)
	if err != nil {
		return nil, fmt.Errorf("eval source: %w", err)
	}

	return value2agg(v)
}

// compiledScript 按源码文本缓存编译结果；编译产物与 VM 实例无关，可跨 VM 复用
func compiledScript(vm *otto.Otto, src interface{}) (*otto.Script, error) {
	wrapped := fmt.Sprintf("(%s)", src)
	if cached, ok := scriptCache.Load(wrapped); ok {
		return cached.(*otto.Script), nil
	}

	script, err := vm.Compile("", wrapped)
	if err != nil {
		return nil, err
	}

	actual, _ := scriptCache.LoadOrStore(wrapped, script)
	return actual.(*otto.Script), nil
}

func value2agg(v otto.Value) (interface{}, error) {
	if v.IsUndefined() {
		// {"$undefined": true}
		return primitive.Undefined{}, nil
	}

	if v.IsPrimitive() {
		return v.Export()
	}

	class := v.Class()
	switch class {
	case Function:
		// {"$code": string}
		return primitive.JavaScript(v.String()), nil

	case Object:
		return object2agg(v.Object())

	case Array:
		return array2agg(v.Object())

	case RegExp:
		return regexp2agg(v.Object())

	default:
		log.Warnf("unexpected object class %s", class)
		return v.Export()
	}
}

func regexp2agg(obj *otto.Object) (interface{}, error) {
	srcField, err := obj.Get("source")
	if err != nil {
		return nil, err
	}

	source, err := srcField.ToString()
	if err != nil {
		return nil, err
	}

	flags := make([]byte, 0, 3)
	has, err := field2boolean(obj, "global")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'g')
	}

	has, err = field2boolean(obj, "ignoreCase")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'i')
	}

	has, err = field2boolean(obj, "multiline")
	if err != nil {
		return nil, err
	}

	if has {
		flags = append(flags, 'm')
	}

	return primitive.Regex{
		Pattern: source,
		Options: string(flags),
	}, nil
}

func field2boolean(obj *otto.Object, field string) (bool, error) {
	v, err := obj.Get(field)
	if err != nil {
		return false, err
	}

	return v.ToBoolean()
}

func object2agg(obj *otto.Object) (interface{}, error) {
	keys := obj.Keys()
	out := make(bson.D, 0, len(keys))
	for _, key := range keys {
		field, err := obj.Get(key)
		if err != nil {
			return nil, fmt.Errorf("get object field %s: %w", key, err)
		}

		val, err := value2agg(field)
		if err != nil {
			return nil, fmt.Errorf("export value for object field %s: %w", key, err)
		}

		out = append(out, bson.E{
			Key:   key,
			Value: val,
		})
	}

	return out, nil
}

func array2agg(a *otto.Object) (interface{}, error) {
	keys := a.Keys()
	out := make([]interface{}, 0, len(keys))
	for i := range keys {
		ele, err := a.Get(keys[i])
		if err != nil {
			return nil, fmt.Errorf("get #%d element in array: %w", i, err)
		}

		val, err := value2agg(ele)
		if err != nil {
			return nil, fmt.Errorf("export value for #%d element in array: %w", i, err)
		}

		out = append(out, val)
	}

	return out, nil
}
