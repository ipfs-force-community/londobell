db.ActorBalance.createIndex({"Addresses":1}, {"sparse": true});
db.ActorBalance.createIndex({"Code":1}, {"sparse": true});
db.ActorBalance.createIndex({"Addresses":1,"Code":1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});
db.ActorBalance.createIndex({"Epoch":1,"Code":1}, {"sparse": true});

db.ActorState.createIndex({"Epoch":1,"Code":1,"Addr":1}, {"sparse": true});

db.AllocatedSectors.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Allocations.createIndex({"Client":1}, {"sparse": true});
db.Allocations.createIndex({"Provider":1}, {"sparse": true});
db.Allocations.createIndex({"Data":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Client":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Provider":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Client":1,"AllocationID":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Provider":1,"AllocationID":1}, {"sparse": true});

db.BlockHeader.createIndex({"Epoch":1,"Miner":1}, {"sparse": true});

db.BlockMessage.createIndex({"Epoch":1,"_id":1}, {"sparse": true});
db.BlockMessage.createIndex({"Epoch":1,"Messages":1}, {"sparse": true});

db.ClaimedPower.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Claims.createIndex({"Provider":1}, {"sparse": true});
db.Claims.createIndex({"Client":1}, {"sparse": true});
db.Claims.createIndex({"Data":1}, {"sparse": true});
db.Claims.createIndex({"Epoch":1,"Provider":1}, {"sparse": true});
db.Claims.createIndex({"Epoch":1,"Client":1}, {"sparse": true});
db.Claims.createIndex({"Epoch":1,"Provider":1,"ClaimID":1}, {"sparse": true});
db.Claims.createIndex({"Epoch":1,"Client":1,"ClaimID":1}, {"sparse": true});

db.DatacapAllowances.createIndex({"Owner":1}, {"sparse": true});
db.DatacapAllowances.createIndex({"Epoch":1,"Owner":1}, {"sparse": true});
db.DatacapAllowances.createIndex({"Epoch":1,"Owner":1,"Operator":1}, {"sparse": true});

db.DatacapBalances.createIndex({"Owner":1}, {"sparse": true});
db.DatacapBalances.createIndex({"Epoch":1,"Owner":1}, {"sparse": true});

db.DealProposal.createIndex({"Epoch":1,"VerifiedDeal":1}, {"sparse": true});
db.DealProposal.createIndex({"VerifiedDeal":1}, {"sparse": true});
db.DealProposal.createIndex({"Provider":1}, {"sparse": true});
db.DealProposal.createIndex({"Client":1}, {"sparse": true});

db.DealProposalDetail.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.DealProposalSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.ExecGas.createIndex({"Epoch":1}, {"sparse": true});

db.ExecTrace.createIndex({"Epoch":1,"Msg.To":1,"Msg.Method":1,"MsgRct.ExitCode":1}, {"sparse": true});
db.ExecTrace.createIndex({"Epoch":1,"Msg.To":1,"Seq":1}, {"sparse": true});
db.ExecTrace.createIndex({"Epoch":1,"Depth":1}, {"sparse": true});
db.ExecTrace.createIndex({"Cid":1}, {"sparse": true});
db.ExecTrace.createIndex({"SignedCid":1}, {"sparse": true});

// no indexes for FilSupply

db.FinalHeight.createIndex({"Cids":1}, {"sparse": true});

db.MarketFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Message.createIndex({"From":1,"Nonce":1}, {"sparse": true});
db.Message.createIndex({"To":1,"Method":1}, {"sparse": true});
db.Message.createIndex({"Detail.Method":1,"Detail.Actor":1}, {"sparse": true});
db.Message.createIndex({"Detail.PackedHeight":1}, {"sparse": true});
db.Message.createIndex({"Detail.PackedHeight":1,"Detail.Method":1}, {"sparse": true});
db.Message.createIndex({"SignedCid":1}, {"sparse": true});

db.MinerDealSector.createIndex({"Epoch":1,"Miner":1}, {"sparse": true});

db.MinerFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MinerSectorSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MiningProfitability.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MultisigBalance.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.PendingTxns.createIndex({"Addr":1}, {"sparse": true});
db.PendingTxns.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});
db.PendingTxns.createIndex({"Epoch":1,"Addr":1,"Detail.TxnID":1}, {"sparse": true});

db.Tipset.createIndex({"ChildEpoch":1}, {"sparse": true});

db.VerifiedRegistry.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

