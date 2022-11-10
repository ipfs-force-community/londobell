package reg

import (
	"fmt"
	"testing"
)

func TestNewExternalActorRegistry(t *testing.T) {
	inv := NewExternalActorRegistry()
	fmt.Printf("inv: %+v\n", inv)
}

//&{map[]
//map[bafk2bzaceaodxeisgr2gcj6o54urg3xddcqnhdrq7hwldpbbty7awstq6zucw:map[0:{Send *abi.EmptyValue *abi.EmptyValue} 1:{Constructor *abi.EmptyValue *abi.EmptyValue} 2:{Create *eam.CreateParams *eam.CreateReturn} 3:{Create2 *eam.Create2Params *eam.Create2Return}]
//bafk2bzacecau3tohdilfx66pohfqdrngpuqd5oew2j5iv3c7sjlrkcm5npqos:map[0:{Send *abi.EmptyValue *abi.EmptyValue}] bafk2bzacecj4dxzkxnzepno5t4q5l4wjl7e2rsoevyizzflugaocdfqwdbco6:map[0:{Send *abi.EmptyValue *abi.EmptyValue} 1:{Constructor *evm.ConstructorParams *abi.EmptyValue} 2:{InvokeContract uint64 []uint8} 3:{GetBytecode *abi.EmptyValue cid.Cid} 4:{GetStorageAt *evm.GetStorageAtParams uint256.Int} 5:{InvokeContractReadOnly uint64 []uint8} 6:{InvokeContractDelegate uint64 []uint8}]]}
