package aggregators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	monitor "github.com/ipfs-force-community/londobell-aggregators/pool-monitor"

	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/londobell/common"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/types/ethtypes"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/account"
	_init "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/store"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	multiquery "github.com/ipfs-force-community/londobell/cmd/londobell-api/multi-query"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/buildnet"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/fullnode"

	logging "github.com/ipfs/go-log/v2"
)

var (
	log = logging.Logger("aggregators")

	RobustMap = make(map[string]string) // ID: robust
	RLock     sync.RWMutex

	AllDealCountMap = make(map[abi.ChainEpoch]int64)
	DLock           sync.RWMutex

	DealsByAddrCountMap = make(map[string]map[abi.ChainEpoch]int64) // ID: {epoch: count}
	DALock              sync.RWMutex

	ActorIDMap = make(map[string]string) // robust: ID
	ALock      sync.RWMutex
)

// todo: 有无必要加缓存？
func GetIDByAddr(ctx context.Context, addrStr string) (string, error) {
	addr, err := address.NewFromString(buildnet.NetPrefix + addrStr)
	if err != nil {
		return "", err
	}

	switch addr.Protocol() {
	case address.ID:
		return addrStr, nil
	case address.SECP256K1, address.Actor, address.BLS, address.Delegated:
		ALock.RLock()
		if actorID, ok := ActorIDMap[addrStr]; ok {
			defer ALock.RUnlock()
			return actorID, nil
		}

		ALock.RUnlock()

		api := fullnode.API.GetAppropriateAPI()
		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return "", err
		}

		ALock.Lock()
		ActorIDMap[addrStr] = ID.String()[1:]
		ALock.Unlock()

		return ID.String()[1:], nil
	default:
		err = fmt.Errorf("invalid addr %v", addrStr)
		return "", err
	}
}

// GetAddrs get [ID,robust] from ActorBalance
func GetAddrs(ctx context.Context, addr string) (model.AddressRes, error) {
	formal := multiquery.DBStateManager.GetFormalCfg()
	cols, ok := multiquery.DBStateManager.GetDBCollections(formal.Url())
	if !ok {
		return model.AddressRes{}, fmt.Errorf("url %v not found in DBCollectionsMap", formal.Url())
	}

	var addressRes []model.AddressRes

	pipe, err := util.Parse(model.Ctx{Addr: addr}, string(addressAggregator))
	if err != nil {
		return model.AddressRes{}, err
	}

	for _, col := range cols.Cols {
		if col != nil && col.Name() == "ActorBalance" {
			cur, err := col.Aggregate(ctx, pipe)
			if err != nil {
				return model.AddressRes{}, err
			}

			err = cur.All(ctx, &addressRes)
			if err != nil {
				return model.AddressRes{}, err
			}

			if len(addressRes) != 1 {
				return model.AddressRes{}, ErrNotFound
			}

			return addressRes[0], nil
		}
	}

	return model.AddressRes{}, fmt.Errorf("no table ActorBalance")
}

func GetRobustByID(ctx context.Context, api v0api.FullNode, IDAddr address.Address, actor *types.Actor) (string, error) {
	if IDAddr.Protocol() != address.ID {
		// warn
		return "", nil
	}

	actorID := IDAddr.String()[1:]

	// find from RobustMap
	RLock.RLock()
	robust, ok := RobustMap[actorID]
	RLock.RUnlock()
	if ok {
		return robust, nil
	}

	res, err := GetAddrs(ctx, actorID)
	if err != nil && err != ErrNotFound {
		return "", err
	}

	if err == nil {
		for _, addr := range res.Addresses {
			if addr == "" {
				continue
			}
			if addr[0] == '1' || addr[0] == '2' || addr[0] == '3' {
				RLock.Lock()
				defer RLock.Unlock()
				RobustMap[actorID] = addr
				return addr, nil
			}
		}

		// 该actor没有robust地址
		return "", nil
	}

	// not found，则为近高度新增的actor
	// account
	if builtin.IsAccountActor(actor.Code) {
		st, err := account.Load(store.ActorStore(ctx, blockstore.NewAPIBlockstore(api)), actor)
		if err != nil {
			return "", err
		}

		robust, err := st.PubkeyAddress()
		if err != nil {
			return "", err
		}

		RLock.Lock()
		RobustMap[actorID] = robust.String()[1:]
		RLock.Unlock()

		return robust.String()[1:], nil
	}

	// other
	iact, err := api.StateGetActor(ctx, _init.Address, types.EmptyTSK)
	if err != nil {
		return "", err
	}

	ist, err := _init.Load(store.ActorStore(ctx, blockstore.NewAPIBlockstore(api)), iact)
	if err != nil {
		return "", err
	}

	var robustStr string
	err = ist.ForEachActor(func(id abi.ActorID, robust address.Address) error {
		idAddr, err := address.NewIDAddress(uint64(id))
		if err != nil {
			return err
		}

		if idAddr.String()[1:] == actorID {
			RLock.Lock()
			RobustMap[actorID] = robust.String()[1:]
			RLock.Unlock()

			robustStr = robust.String()[1:]
			return nil
		}

		return fmt.Errorf("id %v not found", actorID)
	})

	if err != nil {
		return "", err
	}

	return robustStr, nil
}

// GetAllAddrs get [ID, robust, delegated]
func GetAllAddrs(ctx context.Context, addrStr string, api v0api.FullNode) ([]string, error) {
	addrs := make([]string, 0)

	addr, err := address.NewFromString(buildnet.NetPrefix + addrStr)
	if err != nil {
		return nil, err
	}

	switch addr.Protocol() {
	case address.ID:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		if actor.Address != nil {
			delegated := actor.Address
			addrs = append(addrs, delegated.String()[1:])
		}

		robust, err := GetRobustByID(ctx, api, addr, actor)
		if err != nil {
			return nil, err
		}

		if robust != "" {
			addrs = append(addrs, robust)
		}

		return addrs, nil
	case address.SECP256K1, address.Actor, address.BLS:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		if actor.Address != nil {
			delegated := actor.Address
			addrs = append(addrs, delegated.String()[1:])
		}

		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		addrs = append(addrs, ID.String()[1:])

		return addrs, nil
	case address.Delegated:
		addrs = append(addrs, addr.String()[1:])

		actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		ID, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		robust, err := GetRobustByID(ctx, api, ID, actor)
		if err != nil {
			return nil, err
		}

		if robust != "" {
			addrs = append(addrs, robust)
		}

		return addrs, nil
	default:
		return nil, fmt.Errorf("invalid addr: %v", addrStr)
	}
}

func GetTraceByCid(ctx context.Context, cid string) ([]model.TraceForMessageRes, error) {
	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		return nil, err
	}

	pipe, err := util.Parse(model.Ctx{Cid: cid}, string(traceForMessageAggregator))
	if err != nil {
		return nil, err
	}

	var traceForMessageRes []model.TraceForMessageRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "ExecTrace")
		if err != nil {
			return nil, err
		}

		if len(multiResult) == 0 {
			return nil, nil
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawByte, &traceForMessageRes)
		if err != nil {
			return nil, err
		}
	}

	return traceForMessageRes, nil
}

func IsSigTypeDelegatedMessage(from address.Address) bool {
	return from.Protocol() == address.Delegated
}

func newEthTxFromSignedMessage(ctx context.Context, msg *types.Message, signedCid cid.Cid, api v0api.FullNode) (ethtypes.EthTx, error) {
	var tx ethtypes.EthTx
	var err error

	// This is an eth tx
	if msg.From.Protocol() == address.Delegated {
		tx, err = EthTxFromSignedEthMessage(msg)
		if err != nil {
			return ethtypes.EthTx{}, xerrors.Errorf("failed to convert from signed message: %w", err)
		}

		// todo:暂时不计算
		//tx.Hash, err = tx.TxHash()
		//if err != nil {
		//	return ethtypes.EthTx{}, xerrors.Errorf("failed to calculate hash for ethTx: %w", err)
		//}

		fromAddr, err := lookupEthAddress(ctx, msg.From, api)
		if err != nil {
			return ethtypes.EthTx{}, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
		}

		tx.From = fromAddr
	} else if msg.From.Protocol() == address.SECP256K1 { // Secp Filecoin Message
		tx = ethTxFromNativeMessage(ctx, msg, api)
		tx.Hash, err = ethtypes.EthHashFromCid(signedCid)
		if err != nil {
			return tx, err
		}
	} else { // BLS Filecoin message
		tx = ethTxFromNativeMessage(ctx, msg, api)
		tx.Hash, err = ethtypes.EthHashFromCid(signedCid)
		if err != nil {
			return tx, err
		}
	}

	return tx, nil
}

func ethTxFromNativeMessage(ctx context.Context, msg *types.Message, api v0api.FullNode) ethtypes.EthTx {
	// We don't care if we error here, conversion is best effort for non-eth transactions
	from, _ := lookupEthAddress(ctx, msg.From, api)
	to, _ := lookupEthAddress(ctx, msg.To, api)
	return ethtypes.EthTx{
		To:                   &to,
		From:                 from,
		Nonce:                ethtypes.EthUint64(msg.Nonce),
		ChainID:              ethtypes.EthUint64(build.Eip155ChainId),
		Value:                ethtypes.EthBigInt(msg.Value),
		Type:                 ethtypes.Eip1559TxType,
		Gas:                  ethtypes.EthUint64(msg.GasLimit),
		MaxFeePerGas:         ethtypes.EthBigInt(msg.GasFeeCap),
		MaxPriorityFeePerGas: ethtypes.EthBigInt(msg.GasPremium),
		AccessList:           []ethtypes.EthHash{},
	}
}

func lookupEthAddress(ctx context.Context, addr address.Address, api v0api.FullNode) (ethtypes.EthAddress, error) {
	// BLOCK A: We are trying to get an actual Ethereum address from an f410 address.
	// Attempt to convert directly, if it's an f4 address.
	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err == nil && !ethAddr.IsMaskedID() {
		return ethAddr, nil
	}

	// Lookup on the target actor and try to get an f410 address.
	if actor, err := api.StateGetActor(ctx, addr, types.EmptyTSK); err != nil {
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
	idAddrStr, err := GetIDByAddr(ctx, addr.String()[1:])
	if err != nil {
		return ethtypes.EthAddress{}, err
	}

	idAddr, err := address.NewFromString(buildnet.NetPrefix + idAddrStr)
	if err != nil {
		return ethtypes.EthAddress{}, err
	}
	return ethtypes.EthAddressFromFilecoinAddress(idAddr)
}

func EthTxFromSignedEthMessage(msg *types.Message) (ethtypes.EthTx, error) {
	if !IsSigTypeDelegatedMessage(msg.From) {
		return ethtypes.EthTx{}, xerrors.Errorf("signature is not delegated type, from is type: %d", msg.From.Protocol())
	}

	txArgs, err := ethtypes.EthTxArgsFromUnsignedEthMessage(msg)
	if err != nil {
		return ethtypes.EthTx{}, xerrors.Errorf("failed to convert the unsigned message: %w", err)
	}

	// todo: 目前数据库没存Signature，有无必要？
	//r, s, v, err := RecoverSignature(smsg.Signature)
	//if err != nil {
	//	return EthTx{}, xerrors.Errorf("failed to recover signature: %w", err)
	//}

	tx := ethtypes.EthTx{
		Nonce:                ethtypes.EthUint64(txArgs.Nonce),
		ChainID:              ethtypes.EthUint64(txArgs.ChainID),
		To:                   txArgs.To,
		Value:                ethtypes.EthBigInt(txArgs.Value),
		Type:                 ethtypes.Eip1559TxType,
		Gas:                  ethtypes.EthUint64(txArgs.GasLimit),
		MaxFeePerGas:         ethtypes.EthBigInt(txArgs.MaxFeePerGas),
		MaxPriorityFeePerGas: ethtypes.EthBigInt(txArgs.MaxPriorityFeePerGas),
		AccessList:           []ethtypes.EthHash{},
		//V:                    v,
		//R:                    r,
		//S:                    s,
		Input: txArgs.Input,
	}

	// todo: v、R、S为空无法hash
	//tx.Hash, err = tx.TxHash()
	//if err != nil {
	//	return ethtypes.EthTx{}, err
	//}

	//fromAddr, err := lookupEthAddress(ctx, smsg.Message.From, sa)
	//if err != nil {
	//	return ethtypes.EthTx{}, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
	//}
	//
	//tx.From = fromAddr

	return tx, nil
}

func GetMessageByTrace(trace model.TraceForMessageRes) (*types.Message, error) {
	var msg = &types.Message{}

	msg.Version = trace.Version

	from, err := address.NewFromString(buildnet.NetPrefix + trace.From)
	if err != nil {
		return nil, err
	}

	msg.From = from

	to, err := address.NewFromString(buildnet.NetPrefix + trace.To)
	if err != nil {
		return nil, err
	}

	msg.To = to

	methodInfo, err := util.LookupMethodInfo(trace.Epoch, abi.MethodNum(trace.MethodNum), trace.From, trace.To, trace.Actor)
	if err != nil {
		return nil, err
	}

	if !trace.ParamsBson.IsZero() {
		params := methodInfo.ParamObj()
		if params != nil {
			err = params.UnmarshalCBOR(bytes.NewBuffer(trace.ParamsBson.Data))
			if err != nil {
				return nil, err
			}

			buf := new(bytes.Buffer)
			err = params.MarshalCBOR(buf)
			if err != nil {
				return nil, err
			}
			msg.Params = buf.Bytes()
		}
	}

	//params := trace.Params.(map[string]interface{})
	//msg.Params = params["Data"].([]byte)

	msg.Value = big.MustFromString(trace.Value)

	msg.Method = abi.MethodNum(trace.MethodNum)

	msg.Nonce = trace.Nonce

	msg.GasFeeCap = big.MustFromString(trace.GasFeeCap)
	msg.GasPremium = big.MustFromString(trace.GasPremium)
	msg.GasLimit = trace.GasLimit

	return msg, nil
}

func GetTransactionIndexBySeq(seq []uint64) (uint64, error) {
	if len(seq) == 0 {
		return 0, fmt.Errorf("invalid length of seq: %v", len(seq))
	}

	return seq[len(seq)-1], nil
}

func GetParentTipSetByEpoch(ctx context.Context, epoch abi.ChainEpoch) (model.TipSetRes, error) {
	curEpoch := common.GetCurEpoch()

	countUtils, err := multiquery.GetEpochRange(ctx, &multiquery.DBStateManager, curEpoch)
	if err != nil {
		return model.TipSetRes{}, err
	}

	var parentTipSetRes []model.TipSetRes
	// multi dbs query
	{
		multiResult, err := multiquery.MultiRangeQuery(ctx, int64(epoch), int64(epoch)+1, countUtils, parentTipSetAggregator, model.CommonReq{}, "Tipset")
		if err != nil {
			return model.TipSetRes{}, err
		}

		if len(multiResult) == 0 {
			return model.TipSetRes{}, nil
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return model.TipSetRes{}, nil
		}

		err = json.Unmarshal(rawByte, &parentTipSetRes)
		if err != nil {
			return model.TipSetRes{}, err
		}
	}

	if len(parentTipSetRes) != 1 {
		return model.TipSetRes{}, fmt.Errorf("invalid length of parentTipSetRes: %v", len(parentTipSetRes))
	}

	return parentTipSetRes[0], nil
}

func ParseTipSetKey(cidStrs []string) (types.TipSetKey, error) {
	var cids []cid.Cid
	for _, s := range cidStrs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return types.EmptyTSK, err
		}

		cids = append(cids, c)
	}

	return types.NewTipSetKey(cids...), nil
}

func newEthTxFromMessageLookup(ctx context.Context, epoch abi.ChainEpoch, msg *types.Message, signedCid cid.Cid, txIdx uint64, api v0api.FullNode) (ethtypes.EthTx, error) {
	parentTsRes, err := GetParentTipSetByEpoch(ctx, epoch)
	if err != nil {
		return ethtypes.EthTx{}, err
	}

	parentTsKey, err := ParseTipSetKey(parentTsRes.Cids)
	if err != nil {
		return ethtypes.EthTx{}, err
	}

	parentTsCid, err := parentTsKey.Cid()
	if err != nil {
		return ethtypes.EthTx{}, err
	}

	blkHash, err := ethtypes.EthHashFromCid(parentTsCid)
	if err != nil {
		return ethtypes.EthTx{}, err
	}

	tx, err := newEthTxFromSignedMessage(ctx, msg, signedCid, api)
	if err != nil {
		return ethtypes.EthTx{}, err
	}

	var (
		bn = ethtypes.EthUint64(parentTsRes.Epoch)
		ti = ethtypes.EthUint64(txIdx)
	)

	tx.ChainID = ethtypes.EthUint64(build.Eip155ChainId)
	tx.BlockHash = &blkHash
	tx.BlockNumber = &bn
	tx.TransactionIndex = &ti
	return tx, nil
}

func newEthTxReceipt(ctx context.Context, tx ethtypes.EthTx, trace model.TraceForMessageRes, events []types.Event, sa v0api.FullNode) (api.EthTxReceipt, error) {
	var (
		transactionIndex ethtypes.EthUint64
		blockHash        ethtypes.EthHash
		blockNumber      ethtypes.EthUint64
	)

	if tx.TransactionIndex != nil {
		transactionIndex = *tx.TransactionIndex
	}
	if tx.BlockHash != nil {
		blockHash = *tx.BlockHash
	}
	if tx.BlockNumber != nil {
		blockNumber = *tx.BlockNumber
	}

	receipt := api.EthTxReceipt{
		TransactionHash:  tx.Hash,
		From:             tx.From,
		To:               tx.To,
		TransactionIndex: transactionIndex,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Type:             ethtypes.EthUint64(2),
		Logs:             []ethtypes.EthLog{}, // empty log array is compulsory when no logs, or libraries like ethers.js break
		LogsBloom:        ethtypes.EmptyEthBloom[:],
	}

	if exitcode.ExitCode(trace.ExitCode).IsSuccess() {
		receipt.Status = 1
	}
	if exitcode.ExitCode(trace.ExitCode).IsError() {
		receipt.Status = 0
	}

	receipt.GasUsed = ethtypes.EthUint64(big.MustFromString(trace.GasCost.GasUsed).Int64())

	// TODO: handle CumulativeGasUsed
	receipt.CumulativeGasUsed = ethtypes.EmptyEthInt

	effectiveGasPrice := big.Div(big.MustFromString(trace.GasCost.TotalCost), big.MustFromString(trace.GasCost.GasUsed))
	receipt.EffectiveGasPrice = ethtypes.EthBigInt(effectiveGasPrice)

	if receipt.To == nil && exitcode.ExitCode(trace.ExitCode).IsSuccess() {
		// Create and Create2 return the same things.
		var ret eam.CreateExternalReturn
		if err := ret.UnmarshalCBOR(bytes.NewReader(trace.ReturnsBson.Data)); err != nil {
			return api.EthTxReceipt{}, xerrors.Errorf("failed to parse contract creation result: %w", err)
		}
		addr := ethtypes.EthAddress(ret.EthAddress)
		receipt.ContractAddress = &addr
	}

	if len(events) > 0 {
		receipt.Logs = make([]ethtypes.EthLog, 0, len(events))
		for i, evt := range events {
			l := ethtypes.EthLog{
				Removed:          false,
				LogIndex:         ethtypes.EthUint64(i),
				TransactionHash:  tx.Hash,
				TransactionIndex: transactionIndex,
				BlockHash:        blockHash,
				BlockNumber:      blockNumber,
			}

			data, topics, ok := ethLogFromEvent(evt.Entries)
			if !ok {
				// not an eth event.
				continue
			}
			for _, topic := range topics {
				ethtypes.EthBloomSet(receipt.LogsBloom, topic[:])
			}
			l.Data = data
			l.Topics = topics

			addr, err := address.NewIDAddress(uint64(evt.Emitter))
			if err != nil {
				return api.EthTxReceipt{}, xerrors.Errorf("failed to create ID address: %w", err)
			}

			l.Address, err = lookupEthAddress(ctx, addr, sa)
			if err != nil {
				return api.EthTxReceipt{}, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
			}

			ethtypes.EthBloomSet(receipt.LogsBloom, l.Address[:])
			receipt.Logs = append(receipt.Logs, l)
		}
	}

	return receipt, nil
}

func ethLogFromEvent(entries []types.EventEntry) (data []byte, topics []ethtypes.EthHash, ok bool) {
	var (
		topicsFound      [4]bool
		topicsFoundCount int
		dataFound        bool
	)
	for _, entry := range entries {
		// Drop events with non-raw topics to avoid mistakes.
		if entry.Codec != cid.Raw {
			log.Warnw("did not expect an event entry with a non-raw codec", "codec", entry.Codec, "key", entry.Key)
			return nil, nil, false
		}
		// Check if the key is t1..t4
		if len(entry.Key) == 2 && "t1" <= entry.Key && entry.Key <= "t4" {
			// '1' - '1' == 0, etc.
			idx := int(entry.Key[1] - '1')

			// Drop events with mis-sized topics.
			if len(entry.Value) != 32 {
				log.Warnw("got an EVM event topic with an invalid size", "key", entry.Key, "size", len(entry.Value))
				return nil, nil, false
			}

			// Drop events with duplicate topics.
			if topicsFound[idx] {
				log.Warnw("got a duplicate EVM event topic", "key", entry.Key)
				return nil, nil, false
			}
			topicsFound[idx] = true
			topicsFoundCount++

			// Extend the topics array
			for len(topics) <= idx {
				topics = append(topics, ethtypes.EthHash{})
			}
			copy(topics[idx][:], entry.Value)
		} else if entry.Key == "d" {
			// Drop events with duplicate data fields.
			if dataFound {
				log.Warnw("got duplicate EVM event data")
				return nil, nil, false
			}

			dataFound = true
			data = entry.Value
		} else {
			// Skip entries we don't understand (makes it easier to extend things).
			// But we warn for now because we don't expect them.
			log.Warnw("unexpected event entry", "key", entry.Key)
		}

	}

	// Drop events with skipped topics.
	if len(topics) != topicsFoundCount {
		log.Warnw("EVM event topic length mismatch", "expected", len(topics), "actual", topicsFoundCount)
		return nil, nil, false
	}
	return data, topics, true
}

func GetEventsByRoot(ctx context.Context, root string) ([]types.Event, error) {
	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		return nil, err
	}

	pipe, err := util.Parse(model.Ctx{Cid: root}, monitor.GetEventsRootAggregator())
	if err != nil {
		return nil, err
	}

	var eventsRootRes []model.EventsRootRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "EventsRoot")
		if err != nil {
			return nil, err
		}

		if len(multiResult) == 0 {
			return nil, ErrNotFound
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(rawByte, &eventsRootRes)
		if err != nil {
			return nil, err
		}
	}

	if len(eventsRootRes) == 0 {
		return nil, ErrNotFound
	}

	var events []types.Event
	err = json.Unmarshal(eventsRootRes[0].Events.Data, &events)
	//events, err := types.DecodeEvents(eventsRootRes[0].Events.Data)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func GetCidFromEthHash(ctx context.Context, hash string) (string, error) {
	txHash, err := ethtypes.ParseEthHash(hash)
	if err != nil {
		return "", err
	}

	// f4 & other

	countUtils, err := multiquery.GetColsOnly(&multiquery.DBStateManager)
	if err != nil {
		return "", err
	}

	pipe, err := util.Parse(model.Ctx{Cid: hash}, string(monitor.GetMessageCidByHashAggregator()))
	if err != nil {
		return "", err
	}

	var getCidByHashRes []model.MessageCidByHashRes

	// multi dbs query
	{
		multiResult, err := multiquery.MultiTraversalQuery(ctx, pipe, countUtils, "EthHash")
		if err != nil {
			return "", err
		}

		if len(multiResult) == 0 {
			return txHash.ToCid().String(), nil
		}

		raw := multiResult
		rawByte, err := json.Marshal(raw)
		if err != nil {
			return "", err
		}

		err = json.Unmarshal(rawByte, &getCidByHashRes)
		if err != nil {
			return "", err
		}
	}

	if len(getCidByHashRes) == 0 {
		return txHash.ToCid().String(), nil
	}

	return getCidByHashRes[0].Cid, nil

}
