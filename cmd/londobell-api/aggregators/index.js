// equality --> sort --> range
db.ActorBalance.createIndex({"Addresses":1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch": 1, "Addr": 1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch": 1, "Addr": 1, "Balance": 1}, {"sparse": true}); // 待定
db.ActorBalance.createIndex({"Epoch": 1, "Balance": 1}, {"sparse": true}); // 待定
db.ActorBalance.createIndex({"Epoch": 1, "Balance": -1, "Code": 1}, {"sparse": true}); // 待定

db.ActorState.createIndex({"Epoch":1, "Code":1, "Addr":1}, {"sparse": true});

db.ExecTrace.createIndex({"Epoch": 1, "Depth": 1, "Msg.From": 1},{"sparse": true}); // Msg.To
db.ExecTrace.createIndex({"Msg.To":1, "Msg.Method":1, "Detail.Return.RobustAddress":1}, {"sparse": true});
db.ExecTrace.createIndex({"Depth":1, "Msg.To":1, "Epoch":1, "Msg.Method": 1},{"sparse": true});
db.ExecTrace.createIndex({"MsgRct.ExitCode":1, "Epoch":1},{"sparse": true});
db.ExecTrace.createIndex({"Depth":1, "Cid":1, "SignedCid": 1, "Msg.From": 1},{"sparse": true});
// db.ExecTrace.createIndex({"MsgRct.ExitCode":1, "Epoch":1, "Msg.From":1, "Msg.To":1}, {"sparse": true});

db.MinerFunds.createIndex({"Epoch":1, "Info.Owner": 1, "Addr": 1}, {"sparse": true});

db.BlockHeader.createIndex({"Epoch":1, "Miner":1}, {"sparse": true});

// db.MessageBlock.createIndex({"Epoch":1, "Block":1}, {"sparse": true});

db.BlockMessage.createIndex({"_id":1, "Epoch":1}, {"sparse": true});
db.BlockMessage.createIndex({"Epoch":1, "Messages":1}, {"sparse": true});

db.DealProposal.createIndex({"Epoch":1,"_id":1},{"sparse": true}); //todo:_id
db.DealProposal.createIndex({"Epoch":1, "Client": 1, "Provider": 1}, {"sparse": true});

db.MinerSectorHealth.createIndex({"Epoch":1, "Addr":1}, {"sparse": true});

db.Message.createIndex({"_id":1,"Detail.Method":1}, {"sparse": true});


//// notice: 索引建太多了，可能用到不合适的
// // todo
// exectrace: Epoch, MsgRct.ExitCode
// message



