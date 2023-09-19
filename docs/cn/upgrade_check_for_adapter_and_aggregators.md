#### adapter和aggregators升级检查清单
1. actor及网络版本升级
- 重新生成`londobell/cmd/londobell-api/controller/adapter/types.go`
```
make gen-types
```

- 检查adapter和aggregators中引用到`github.com/filecoin-project/go-state-types/actors`和`github.com/filecoin-project/specs-actors/{{ .Version}}/actors/builtin`等涉及到版本的代码，并相应升级。

2. github.com/filecoin-project/go-address引用  
检查`github.com/filecoin-project/go-address`的地址Protocol是否有增加，如果增加则对aggregators和adapter中涉及到protocol判断的地方进行相应升级。  
例如`https://github.com/ipfs-force-community/londobell/blob/723832a677a0a36c621337afb28a5d1f2f8fcd11/cmd/londobell-api/controller/adapter/actor.go#L79`:
```
if addr.Protocol() == address.ID {
    actorID = addr
} else if addr.Protocol() == address.BLS || addr.Protocol() == address.SECP256K1 || addr.Protocol() == address.Actor || addr.Protocol() == address.Delegated {
    actorID, err = api.StateLookupID(ctx, addr, ts.Key())
    if err != nil {
        alog.Error(err)
        util.ReturnOnErr(c, err)
        return
    }

    if addr.Protocol() == address.Delegated {
        delegatedAddr = addr
    } else {
        actorAddr = addr
    }
}
```

3. github.com/filecoin-project/go-state-types/crypto/signature.go 引用  
检查`github.com/filecoin-project/go-state-types/crypto`的SigType是否有增加，如果增加则对aggregators和adapter中涉及到SigType判断的地方进行相应升级。  
例如`https://github.com/ipfs-force-community/londobell/blob/723832a677a0a36c621337afb28a5d1f2f8fcd11/cmd/londobell-api/controller/adapter/util.go#L23`:
```
if smsg.Signature.Type == crypto.SigTypeDelegated {
    tx, err = ethtypes.EthTxFromSignedEthMessage(smsg)
    if err != nil {
        return ethtypes.EmptyEthHash, xerrors.Errorf("failed to convert from signed message: %w", err)
    }

    tx.Hash, err = tx.TxHash()
    if err != nil {
        return ethtypes.EmptyEthHash, xerrors.Errorf("failed to calculate hash for ethTx: %w", err)
    }

    fromAddr, err := lookupEthAddress(ctx, smsg.Message.From, sa)
    if err != nil {
        return ethtypes.EmptyEthHash, xerrors.Errorf("failed to resolve Ethereum address: %w", err)
    }

    tx.From = fromAddr
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
```

4. 新增方法名  
如果内置actor有新增方法名，应检查`https://github.com/ipfs-force-community/londobell/blob/723832a677a0a36c621337afb28a5d1f2f8fcd11/cmd/londobell-api/controller/adapter/mpool.go#L219`和`https://github.com/ipfs-force-community/londobell/blob/723832a677a0a36c621337afb28a5d1f2f8fcd11/cmd/londobell-api/util/util.go#L31`，看是否需要相应升级

5. 新增内置单例actor  
如果有新增的内置单例actor，应根据该actor添加的网络版本来更新其创建时间。例如：`https://github.com/ipfs-force-community/londobell/blob/be9cbe591bd5b16306cdbc469e7aa544431a0548/cmd/londobell-api/controller/aggregators/create_time.go#L55`

6. 编译失败后检查lotus方法签名是否变化   

7. 新版本投入`calibnet`测试观察结果，若发生未期望的情况，检查逻辑是否适应上一版本  
