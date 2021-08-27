db.AllocatedSectors.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.BlockHeader.createIndex({"Epoch":1,"Miner":1}, {"sparse": true});

db.ClaimedPower.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.DealProposalSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.ExecGas.createIndex({"Epoch":1}, {"sparse": true});

db.ExecTrace.createIndex({"Epoch":1,"Msg.To":1,"Msg.Method":1,"MsgRct.ExitCode":1}, {"sparse": true});
db.ExecTrace.createIndex({"Epoch":1,"Msg.To":1,"Seq":1}, {"sparse": true});
db.ExecTrace.createIndex({"Epoch":1,"Depth":1}, {"sparse": true});
db.ExecTrace.createIndex({"Cid":1}, {"sparse": true});

// no indexes for FilSupply

db.Message.createIndex({"From":1,"Nonce":1}, {"sparse": true});
db.Message.createIndex({"To":1,"Method":1}, {"sparse": true});
db.Message.createIndex({"Detail.Method":1,"Detail.Actor":1}, {"sparse": true});
db.Message.createIndex({"Detail.PackedHeight":1},{"sparse":true});

db.MinerFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MinerSectorSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MiningProfitability.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MultisigBalance.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Tipset.createIndex({"ChildEpoch":1}, {"sparse": true});

db.VerifiedRegistry.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

