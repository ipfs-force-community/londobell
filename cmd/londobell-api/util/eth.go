package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtintypes "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multicodec"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

// The address used in messages to actors that have since been deleted.
//
// 0xff0000000000000000000000ffffffffffffffff
var revertedEthAddress ethtypes.EthAddress

func init() {
	revertedEthAddress[0] = 0xff
	for i := 20 - 8; i < 20; i++ {
		revertedEthAddress[i] = 0xff
	}
}

// node/impl/full/eth_utils.go.ethTxHashFromSignedMessage
func NewEthHashFromSignedMessage(smsg *types.SignedMessage) (ethtypes.EthHash, error) {
	if smsg.Signature.Type == crypto.SigTypeDelegated {
		tx, err := ethtypes.EthTransactionFromSignedFilecoinMessage(smsg)
		if err != nil {
			return ethtypes.EthHash{}, xerrors.Errorf("failed to convert from signed message: %w", err)
		}

		return tx.TxHash()
	} else if smsg.Signature.Type == crypto.SigTypeSecp256k1 {
		return ethtypes.EthHashFromCid(smsg.Cid())
	}
	// else BLS message
	return ethtypes.EthHashFromCid(smsg.Message.Cid())
}

// LookupEthAddress makes its best effort at finding the Ethereum address for a
// Filecoin address. It does the following:
//
//  1. If the supplied address is an f410 address, we return its payload as the EthAddress.
//  2. Otherwise (f0, f1, f2, f3), we look up the actor on the state tree. If it has a delegated address, we return it if it's f410 address.
//  3. Otherwise, we fall back to returning a masked ID Ethereum address. If the supplied address is an f0 address, we
//     use that ID to form the masked ID address.
//  4. Otherwise, we fetch the actor's ID from the state tree and form the masked ID with it.
//
// If the actor doesn't exist in the state-tree but we have its ID, we use a masked ID address. It could have been deleted.
func LookupEthAddress(ctx context.Context, addr address.Address, api v0api.FullNode) (ethtypes.EthAddress, error) {
	// Attempt to convert directly, if it's an f4 address.
	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err == nil && !ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	// Otherwise, resolve the ID addr.
	idAddr, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		return ethtypes.EthAddress{}, err
	}

	// revive:disable:empty-block easier to grok when the cases are explicit

	// Lookup on the target actor and try to get an f410 address.
	if actor, err := api.StateGetActor(ctx, idAddr, types.EmptyTSK); errors.Is(err, types.ErrActorNotFound) {
		// Not found -> use a masked ID address
	} else if err != nil {
		// Any other error -> fail.
		return ethtypes.EthAddress{}, err
	} else if actor.DelegatedAddress == nil {
		// No delegated address -> use masked ID address.
	} else if ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(*actor.DelegatedAddress); err == nil && !ethAddr.IsMaskedID() {
		// Conversable into an eth address, use it.
		return ethAddr, nil
	}

	// Otherwise, use the masked address.
	return ethtypes.EthAddressFromFilecoinAddress(idAddr)
}

// decodePayload is a utility function which decodes the payload using the given codec
func decodePayload(payload []byte, codec uint64) (ethtypes.EthBytes, error) {
	switch multicodec.Code(codec) {
	case multicodec.Identity:
		return nil, nil
	case multicodec.DagCbor, multicodec.Cbor:
		buf, err := cbg.ReadByteArray(bytes.NewReader(payload), uint64(len(payload)))
		if err != nil {
			return nil, xerrors.Errorf("decodePayload: failed to decode cbor payload: %w", err)
		}
		return buf, nil
	case multicodec.Raw:
		return ethtypes.EthBytes(payload), nil
	}

	return nil, xerrors.Errorf("decodePayload: unsupported codec: %d", codec)
}

func encodeFilecoinParamsAsABI(method abi.MethodNum, codec uint64, params []byte) []byte {
	buf := []byte{0x86, 0x8e, 0x10, 0xc4} // Native method selector.
	return append(buf, encodeAsABIHelper(uint64(method), codec, params)...)
}

// Format 2 numbers followed by an arbitrary byte array as solidity ABI. Both our native
// inputs/outputs follow the same pattern, so we can reuse this code.
func encodeAsABIHelper(param1 uint64, param2 uint64, data []byte) []byte {
	const EVM_WORD_SIZE = 32

	// The first two params are "static" numbers. Then, we record the offset of the "data" arg,
	// then, at that offset, we record the length of the data.
	//
	// In practice, this means we have 4 256-bit words back to back where the third arg (the
	// offset) is _always_ '32*3'.
	staticArgs := []uint64{param1, param2, EVM_WORD_SIZE * 3, uint64(len(data))}
	// We always pad out to the next EVM "word" (32 bytes).
	totalWords := len(staticArgs) + (len(data) / EVM_WORD_SIZE)
	if len(data)%EVM_WORD_SIZE != 0 {
		totalWords++
	}
	sz := totalWords * EVM_WORD_SIZE
	buf := make([]byte, sz)
	offset := 0
	// Below, we use copy instead of "appending" to preserve all the zero padding.
	for _, arg := range staticArgs {
		// Write each "arg" into the last 8 bytes of each 32 byte word.
		offset += EVM_WORD_SIZE
		start := offset - 8
		binary.BigEndian.PutUint64(buf[start:offset], arg)
	}

	// Finally, we copy in the data.
	copy(buf[offset:], data)

	return buf
}

// Convert a native message to an eth transaction.
//
//   - The state-tree must be from after the message was applied (ideally the following tipset).
//   - In some cases, the "to" address may be `0xff0000000000000000000000ffffffffffffffff`. This
//     means that the "to" address has not been assigned in the passed state-tree and can only
//     happen if the transaction reverted.
//
// ethTxFromNativeMessage does NOT populate:
// - BlockHash
// - BlockNumber
// - TransactionIndex
// - Hash
func EthTxFromNativeMessage(ctx context.Context, msg *types.Message, api v0api.FullNode) (ethtypes.EthTx, error) {
	// Lookup the from address. This must succeed.
	from, err := LookupEthAddress(ctx, msg.From, api)
	if err != nil {
		return ethtypes.EthTx{}, xerrors.Errorf("failed to lookup sender address %s when converting a native message to an eth txn: %w", msg.From, err)
	}
	// Lookup the to address. If the recipient doesn't exist, we replace the address with a
	// known sentinel address.
	to, err := LookupEthAddress(ctx, msg.To, api)
	if err != nil {
		if !errors.Is(err, types.ErrActorNotFound) {
			return ethtypes.EthTx{}, xerrors.Errorf("failed to lookup receiver address %s when converting a native message to an eth txn: %w", msg.To, err)
		}
		to = revertedEthAddress
	}

	// For empty, we use "0" as the codec. Otherwise, we use CBOR for message
	// parameters.
	var codec uint64
	if len(msg.Params) > 0 {
		codec = uint64(multicodec.Cbor)
	}

	maxFeePerGas := ethtypes.EthBigInt(msg.GasFeeCap)
	maxPriorityFeePerGas := ethtypes.EthBigInt(msg.GasPremium)

	// We decode as a native call first.
	ethTx := ethtypes.EthTx{
		To:                   &to,
		From:                 from,
		Input:                encodeFilecoinParamsAsABI(msg.Method, codec, msg.Params),
		Nonce:                ethtypes.EthUint64(msg.Nonce),
		ChainID:              ethtypes.EthUint64(build.Eip155ChainId),
		Value:                ethtypes.EthBigInt(msg.Value),
		Type:                 ethtypes.EIP1559TxType,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         &maxFeePerGas,
		MaxPriorityFeePerGas: &maxPriorityFeePerGas,
		AccessList:           []ethtypes.EthHash{},
	}

	// Then we try to see if it's "special". If we fail, we ignore the error and keep treating
	// it as a native message. Unfortunately, the user is free to send garbage that may not
	// properly decode.
	if msg.Method == builtintypes.MethodsEVM.InvokeContract {
		// try to decode it as a contract invocation first.
		if inp, err := decodePayload(msg.Params, codec); err == nil {
			ethTx.Input = []byte(inp)
		}
	} else if msg.To == builtin.EthereumAddressManagerActorAddr && msg.Method == builtintypes.MethodsEAM.CreateExternal {
		// Then, try to decode it as a contract deployment from an EOA.
		if inp, err := decodePayload(msg.Params, codec); err == nil {
			ethTx.Input = []byte(inp)
			ethTx.To = nil
		}
	}

	return ethTx, nil
}

func NewEthTxFromMessage(ctx context.Context, msg *types.Message, msgCID cid.Cid, api v0api.FullNode) (ethtypes.EthTx, error) {
	var tx ethtypes.EthTx
	var err error

	// This is an eth tx
	if msg.From.Protocol() == address.Delegated {
		ethTx, err := ethTransactionFromSignedFilecoinMessage(msg)
		if err != nil {
			return ethtypes.EthTx{}, xerrors.Errorf("failed to convert from signed message: %w", err)
		}
		ethTx.Hash, err = ethtypes.EthHashFromCid(msgCID)
		if err != nil {
			return ethtypes.EthTx{}, xerrors.Errorf("failed to convert from signed message: %w", err)
		}
	} else if msg.From.Protocol() == address.SECP256K1 { // Secp Filecoin Message
		tx, err = EthTxFromNativeMessage(ctx, msg, api)
		if err != nil {
			return ethtypes.EthTx{}, err
		}
		tx.Hash, err = ethtypes.EthHashFromCid(msgCID)
		if err != nil {
			return ethtypes.EthTx{}, err
		}
	} else { // BLS Filecoin message
		tx, err = EthTxFromNativeMessage(ctx, msg, api)
		if err != nil {
			return ethtypes.EthTx{}, err
		}
		tx.Hash, err = ethtypes.EthHashFromCid(msgCID)
		if err != nil {
			return ethtypes.EthTx{}, err
		}
	}

	return tx, nil
}

func getEthParamsAndRecipient(msg *types.Message) (params []byte, to *ethtypes.EthAddress, err error) {
	if len(msg.Params) > 0 {
		paramsReader := bytes.NewReader(msg.Params)
		var err error
		params, err = cbg.ReadByteArray(paramsReader, uint64(len(msg.Params)))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read params byte array: %w", err)
		}
		if paramsReader.Len() != 0 {
			return nil, nil, fmt.Errorf("extra data found in params")
		}
		if len(params) == 0 {
			return nil, nil, fmt.Errorf("non-empty params encode empty byte array")
		}
	}

	if msg.To == builtintypes.EthereumAddressManagerActorAddr {
		if msg.Method != builtintypes.MethodsEAM.CreateExternal {
			return nil, nil, fmt.Errorf("unsupported EAM method")
		}
	} else if msg.Method == builtintypes.MethodsEVM.InvokeContract {
		addr, err := ethtypes.EthAddressFromFilecoinAddress(msg.To)
		if err != nil {
			return nil, nil, err
		}
		to = &addr
	} else {
		return nil, nil,
			fmt.Errorf("invalid methodnum %d: only allowed method is InvokeContract(%d) or CreateExternal(%d)",
				msg.Method, builtintypes.MethodsEVM.InvokeContract, builtintypes.MethodsEAM.CreateExternal)
	}

	return params, to, nil
}

func ethTransactionFromSignedFilecoinMessage(msg *types.Message) (*ethtypes.EthTx, error) {
	if msg == nil {
		return nil, errors.New("signed message is nil")
	}

	// Ensure the signature type is delegated.
	if msg.From.Protocol() != address.Delegated {
		return nil, fmt.Errorf("signature is not delegated type, is type: %v", msg.From.Protocol())
	}

	// Convert Filecoin address to Ethereum address.
	_, err := ethtypes.EthAddressFromFilecoinAddress(msg.From)
	if err != nil {
		return nil, fmt.Errorf("sender was not an eth account")
	}

	// Extract Ethereum parameters and recipient from the message.
	params, to, err := getEthParamsAndRecipient(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input params and recipient: %w", err)
	}

	// Check for supported message version.
	if msg.Version != 0 {
		return nil, fmt.Errorf("unsupported msg version: %d", msg.Version)
	}

	from, err := ethtypes.EthAddressFromFilecoinAddress(msg.From)
	if err != nil {
		return &ethtypes.EthTx{}, xerrors.Errorf("sender was not an eth account")
	}

	gasFeeCap := ethtypes.EthBigInt(msg.GasFeeCap)
	gasPremium := ethtypes.EthBigInt(msg.GasPremium)

	ethTx := &ethtypes.EthTx{
		ChainID: ethtypes.EthUint64(build.Eip155ChainId),
		Type:    ethtypes.EIP1559TxType,
		Nonce:   ethtypes.EthUint64(msg.Nonce),
		// Hash:                 hash,
		To:                   to,
		Value:                ethtypes.EthBigInt(msg.Value),
		Input:                params,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         &gasFeeCap,
		MaxPriorityFeePerGas: &gasPremium,
		From:                 from,
		// R:                    EthBigInt(tx.R),
		// S:                    EthBigInt(tx.S),
		// V:                    EthBigInt(tx.V),
	}

	// todo: 目前没有签名
	// Determine the type of transaction based on the signature length
	// switch len(smsg.Signature.Data) {
	// case EthEIP1559TxSignatureLen:
	// 	tx := Eth1559TxArgs{
	// 		ChainID:              build.Eip155ChainId,
	// 		Nonce:                int(smsg.Message.Nonce),
	// 		To:                   to,
	// 		Value:                smsg.Message.Value,
	// 		Input:                params,
	// 		MaxFeePerGas:         smsg.Message.GasFeeCap,
	// 		MaxPriorityFeePerGas: smsg.Message.GasPremium,
	// 		GasLimit:             int(smsg.Message.GasLimit),
	// 	}
	// 	if err := tx.InitialiseSignature(smsg.Signature); err != nil {
	// 		return nil, fmt.Errorf("failed to initialise signature: %w", err)
	// 	}
	// 	return &tx, nil

	// case EthLegacyHomesteadTxSignatureLen, EthLegacy155TxSignatureLen0, EthLegacy155TxSignatureLen1:
	// 	legacyTx := &EthLegacyHomesteadTxArgs{
	// 		Nonce:    int(smsg.Message.Nonce),
	// 		To:       to,
	// 		Value:    smsg.Message.Value,
	// 		Input:    params,
	// 		GasPrice: smsg.Message.GasFeeCap,
	// 		GasLimit: int(smsg.Message.GasLimit),
	// 	}
	// 	// Process based on the first byte of the signature
	// 	switch smsg.Signature.Data[0] {
	// 	case EthLegacyHomesteadTxSignaturePrefix:
	// 		if err := legacyTx.InitialiseSignature(smsg.Signature); err != nil {
	// 			return nil, fmt.Errorf("failed to initialise signature: %w", err)
	// 		}
	// 		return legacyTx, nil
	// 	case EthLegacy155TxSignaturePrefix:
	// 		tx := &EthLegacy155TxArgs{
	// 			legacyTx: legacyTx,
	// 		}
	// 		if err := tx.InitialiseSignature(smsg.Signature); err != nil {
	// 			return nil, fmt.Errorf("failed to initialise signature: %w", err)
	// 		}
	// 		return tx, nil
	// 	default:
	// 		return nil, fmt.Errorf("unsupported legacy transaction; first byte of signature is %d", smsg.Signature.Data[0])
	// 	}

	// default:
	// 	return nil, fmt.Errorf("unsupported signature length")
	// }

	return ethTx, nil
}
