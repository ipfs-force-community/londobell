package adapter

import (
	"context"

	"github.com/filecoin-project/lotus/build"

	"github.com/filecoin-project/lotus/api/v0api"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var (
	log = logging.Logger("adapter")
)

func NewEthHashFromSignedMessage(ctx context.Context, smsg *types.SignedMessage, sa v0api.FullNode) (ethtypes.EthHash, error) {
	var tx ethtypes.EthTx
	var err error

	// This is an eth tx
	if smsg.Signature.Type == crypto.SigTypeDelegated {
		ethTrans, err := ethtypes.EthTransactionFromSignedFilecoinMessage(smsg)
		if err != nil {
			return ethtypes.EmptyEthHash, xerrors.Errorf("failed to convert from signed message: %w", err)
		}

		tx, err = ethTrans.ToEthTx(smsg)
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	} else if smsg.Signature.Type == crypto.SigTypeSecp256k1 { // Secp Filecoin Message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), sa)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	} else { // BLS Filecoin message
		tx = ethTxFromNativeMessage(ctx, smsg.VMMessage(), sa)
		tx.Hash, err = ethtypes.EthHashFromCid(smsg.Message.Cid())
		if err != nil {
			return ethtypes.EmptyEthHash, err
		}
	}

	return tx.Hash, nil
}

func lookupEthAddress(ctx context.Context, addr address.Address, sa v0api.FullNode) (ethtypes.EthAddress, error) {
	// BLOCK A: We are trying to get an actual Ethereum address from an f410 address.
	// Attempt to convert directly, if it's an f4 address.
	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err == nil && !ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	// Lookup on the target actor and try to get an f410 address.
	if actor, err := sa.StateGetActor(ctx, addr, types.EmptyTSK); err != nil {
		return ethtypes.EthAddress{}, err
	} else if actor.Address != nil {
		if ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(*actor.Address); err == nil && !ethAddr.IsMaskedID() {
			return ethAddr, nil
		}
	}

	// BLOCK B: We gave up on getting an actual Ethereum address and are falling back to a Masked ID address.
	// Check if we already have an ID addr, and use it if possible.
	if err == nil && ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	// Otherwise, resolve the ID addr.
	idAddr, err := sa.StateLookupID(ctx, addr, types.EmptyTSK)
	if err != nil {
		return ethtypes.EthAddress{}, err
	}
	return ethtypes.EthAddressFromFilecoinAddress(idAddr)
}

func ethTxFromNativeMessage(ctx context.Context, msg *types.Message, sa v0api.FullNode) ethtypes.EthTx {
	// We don't care if we error here, conversion is best effort for non-eth transactions
	from, _ := lookupEthAddress(ctx, msg.From, sa)
	to, _ := lookupEthAddress(ctx, msg.To, sa)
	maxFeePerGas := ethtypes.EthBigInt(msg.GasFeeCap)
	maxPriorityFeePerGas := ethtypes.EthBigInt(msg.GasPremium)
	return ethtypes.EthTx{
		To:                   &to,
		From:                 from,
		Nonce:                ethtypes.EthUint64(msg.Nonce),
		ChainID:              ethtypes.EthUint64(build.Eip155ChainId),
		Value:                ethtypes.EthBigInt(msg.Value),
		Type:                 ethtypes.EIP1559TxType,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         &maxFeePerGas,
		MaxPriorityFeePerGas: &maxPriorityFeePerGas,
		AccessList:           []ethtypes.EthHash{},
	}
}
