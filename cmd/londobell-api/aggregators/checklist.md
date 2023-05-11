1. 有sort的脚本测试性能，随着表数据增多(分页接口)，可能会超出内存限制
2. 所有接口参照文档，正确形检查一遍
3. 所有接口脚本跑一遍
4. 检查全局变量multiquery.DBStateManager有无冲突
5. multi state备份
6. 检查所有脚本响应速度，是否使用索引


优先满足最常用的接口的索引需求
优化方向： 1. 外层表使用最合适的索引；2. 内联表也建立索引； 3. 裁剪中间过程中的project
// todo:
- MultiPagingQuery相关: 随着数据增多，可能会越来越慢; 分页到后面可能会很慢
1. 大额转账、actor转账()  db.Message.createIndex({"Detail.PackedHeight": -1, "Value": 1}, {"sparse": true})   待Value加入完成
2. actormessage_by_methodname(29s/3s  Depth_1_Msg.From_1_Msg.To_1_Msg.Method_1_Epoch_1)、block(ms)、blockmessages_by_methodname(3s)、messages_for_actor(6s,trace索引？？)   db.ExecTrace.createIndex({"Depth":1, "Epoch":-1, "Msg.From": 1, "Msg.To": 1},{"sparse": true});
3. blockheaders_by_miner 没有使用到索引？？

- totalCount相关
1. RefreshBlockMsgs  7m53.546029338s
2. RefreshBlockMsgsByMethodName  4h34m13.029812671s  countOfBlockMessagesByMethodNameAggregator  Depth_1_Epoch_-1_Msg.From_1_Msg.To_1 和 Depth_1_Msg.From_1_Msg.To_1_Msg.Method_1_Epoch_1  90高度6s
3. RefreshActorMsgsByMethodName  6h28m58.773335516s  countOfActorMessagesByMethodNameAggregator  2s
4. RefreshActorMsgs  14m2.735117765s  allActorsForBlockMessageAggregator  ms
5. RefreshActorTransferMsgs  22h56m42.5278149s  countOfTransfersForActor2Aggregator  ms
6. RefreshMinedMsgsMaps  35.186860441s  minedCountForMinersAggregator  ms
7. RefreshTransfersForLargeAmount  19h28m56.153828691s  countOfLargeAmountTransfersAggregator  10s



- 其他
1. address  db.ActorBalance.createIndex({"Addresses":1}, {"sparse": true});
2. actor_state_epoch  db.ActorState.createIndex({"Epoch": 1, "Addr": 1}, {"sparse": true});
3. balance  db.ActorBalance.createIndex({"Epoch": 1, "Addr": 1}, {"sparse": true});
4. richlist  db.ActorBalance.createIndex({"Epoch": 1, "Balance": -1, "Code": 1}, {"sparse": true}); // 索引待定
5. agg_pre_netfee  db.ExecTrace.createIndex({"Depth":1, "MsgRct.ExitCode":1, "Epoch": 1, "Msg.Method": 1},{"sparse": true}); // 索引待定
6. agg_pro_netfee  db.ExecTrace.createIndex({"Depth":1, "Msg.To":1, "Msg.Method": 1, "Epoch": 1},{"sparse": true}); // 索引待定
7. traces  只需要Epoch:1
8. trace_for_message  db.ExecTrace.createIndex({"Depth":1, "Cid":1, "SignedCid": 1, "Msg.From": 1},{"sparse": true});
9. batch_trace_for_message 比trace_for_message多 Epoch:1
10. child_transfers_for_message  db.ExecTrace.createIndex Depth:1,"Cid":1, "SignedCid": 1,SubCallCount:1
11. multisig_message  
12. miner_blockreward  db.ExecTrace.createIndex Depth:1,Msg.From:1,Msg.To:1,Msg.Method:1,Epoch:1
13. miners_blockreward  db.ExecTrace.createIndex Depth:1,Msg.From:1,Msg.Method:1,Epoch:1
14. miners_mined  db.ExecTrace.createIndex Depth:1,Msg.From:1,Msg.Method:1,Epoch:1
15. wincount  db.ExecTrace.createIndex Depth:1,Msg.From:1,Msg.To:1,Msg.Method:1,Epoch:1
16. total_block_count  db.ExecTrace.createIndex Depth:1,Msg.From:1,Msg.To:1,Msg.Method:1,Epoch:1
17. miners_for_owner  db.MinerFunds.createIndex "Info.Owner": 1, "Epoch":1
18. all_owners  db.MinerFunds.createIndex Epoch: 1
19. miner_info  db.MinerFunds.createIndex Epoch:1,Addr:1
20. miners_info  db.MinerFunds.createIndex Epoch: 1
21. gascost_for_sector  db.ExecTrace.createIndex Depth:1, Msg.Method:1,Epoch:1
22. burn_monitor  db.ExecTrace.createIndex Msg.To:1,Msg.Method:1,Msg.From:1,Epoch:1
23. punishment  db.ExecTrace.createIndex Msg.To:1,Msg.Method:1,SubCallCount:1,Detail.Return:1,GasCost:1,Epoch:1
24. latest_time_of_trace  db.ExecTrace.createIndex Epoch:-1,Msg.From:1,Msg.To:1
25. create_time  db.ExecTrace.createIndex Msg.To:1,Msg.Method:1,Detail.Return.RobustAddress:1
26. deals  db.DealProposal.createIndex "Epoch":1,"_id":1
27. deal_by_id、detail_for_deal   db.DealProposal.createIndex _id:1
28. deals_by_addr
28. blockheader  db.BlockHeader.createIndex Epoch: -1
29. blockheaders_by_miner  Epoch_-1_Miner_1索引失效  db.BlockHeader.createIndex Epoch: -1, Miner: 1
30. minedcount_for_miner 没用到索引
30. blocks_for_message  db.BlockMessage.createIndex Messages:1,Epoch:1
31. count_and_methods_of_messages_for_blockheader  db.BlockMessage.createIndex _id:1
32. messages_for_block  db.BlockMessage.createIndex _id:1
33. blockheader_messages_by_methodname  db.BlockMessage.createIndex _id:1


// 合并索引
agg_pre_netfee  MsgRct.ExitCode_1_Epoch_-1  db.ExecTrace.createIndex({"Depth":1, "Msg.To":1, "MsgRct.ExitCode":1, "Msg.Method": 1, "Epoch": 1},{"sparse": true});
agg_pro_netfee  Depth_1_Epoch_-1_Msg.From_1_Msg.To_1
//trace_for_message、batch_trace_for_message  db.ExecTrace.createIndex({"Depth":1, "Cid":1, "SignedCid": 1, "Msg.From": 1, "Epoch": 1},{"sparse": true});
or trace_for_message、child_transfers_for_message  db.ExecTrace.createIndex({"Depth":1, "Cid":1, "SignedCid": 1, "Msg.From": 1, "SubCallCount":1},{"sparse": true});
batch_trace_for_message
miner_blockreward、miners_blockreward、wincount、gascost_for_sector  Depth_1_Epoch_-1_Msg.From_1_Msg.To_1
miners_mined、total_block_count  db.ExecTrace.createIndex({"Depth":1,"Msg.From":1,"Msg.To":1,"Msg.Method":1,"Epoch":1})
punishment  Epoch_-1_Msg.From_1_Msg.To_1  db.ExecTrace.createIndex({"Msg.To":1,"Msg.Method":1,"SubCallCount":1,"Detail.Return":1,"GasCost":1,"Epoch":1})
latest_time_of_trace、burn_monitor  db.ExecTrace.createIndex({"Epoch":-1,"Msg.From":1,"Msg.To":1}) "Epoch":1 2者只能取其一
create_time  db.ExecTrace.createIndex({"Msg.To":1,"Msg.Method":1,"Detail.Return.RobustAddress":1})  



// db.ExecTrace.createIndex({"Depth":1, "Epoch":-1, "Msg.From": 1, "Msg.To": 1},{"sparse": true}); // todo: 待定，看有没有其他脚本用了？
db.ExecTrace.createIndex({"MsgRct.ExitCode":1, "Epoch":-1},{"sparse": true});
db.ExecTrace.createIndex({"Depth":1, "Msg.To":1, "MsgRct.ExitCode":1, "Msg.Method": 1, "Epoch": 1},{"sparse": true});
db.ExecTrace.createIndex({"Depth":1, "Cid":1, "SignedCid": 1, "Msg.From": 1, "SubCallCount": 1},{"sparse": true});
db.ExecTrace.createIndex({"Depth":1,"Msg.From":1,"Msg.To":1,"Msg.Method":1,"Epoch":1},{"sparse": true});
db.ExecTrace.createIndex({"Epoch":-1,"Msg.From":1,"Msg.To":1},{"sparse": true});
db.ExecTrace.createIndex({"Msg.To":1,"Msg.Method":1,"Detail.Return.RobustAddress":1},{"sparse": true});
db.ExecTrace.createIndex({"Cid":1},{"sparse": true});
db.ExecTrace.createIndex({"SignedCid":1},{"sparse": true});

//db.ExecTrace.createIndex({"Depth":1,"MsgRct.GasUsed":1,"Epoch":1, "Msg.From":1, "Msg.To":1},{"sparse": true});
//db.ExecTrace.createIndex({"Depth":1,"Msg.From":1, "Msg.To":1, "MsgRct.GasUsed":1,"Epoch":1,},{"sparse": true});

db.ExecTrace.createIndex({"MsgRct.ExitCode":1, "Epoch":-1, "Msg.From":1, "Msg.To":1},{"sparse": true});


// db.ExecTrace.createIndex({"Epoch": -1},{"sparse": true});

db.Message.createIndex({"Detail.PackedHeight": 1}, {"sparse": true})
db.Message.createIndex({"Detail.Method": 1}, {"sparse": true})

db.ActorBalance.createIndex({"Addresses":1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch": 1, "Addr": 1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch": 1, "Balance": -1, "Code": 1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch": 1, "Code": 1}, {"sparse": true});

db.ActorState.createIndex({"Epoch": 1, "Addr": 1}, {"sparse": true});

db.MinerFunds.createIndex({"Info.Owner": 1, "Epoch":1}, {"sparse": true});
db.MinerFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MinerSectorHealth.createIndex({"Epoch":1, "Addr":1}, {"sparse": true});

db.DealProposal.createIndex({"Epoch":1,"_id":-1,"Client":1, "Provider":1},{"sparse": true})

db.BlockHeader.createIndex({"Epoch": -1, "Miner": 1}, {"sparse": true});  //索引全部失效

db.BlockMessage.createIndex({"Epoch":1, "Messages":1}, {"sparse": true})


//GasCost不为null 说明是显式消息？？