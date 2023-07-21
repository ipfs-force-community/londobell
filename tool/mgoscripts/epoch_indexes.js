db.ActorAddress.createIndex({"RobustAddress":1}, {"sparse": true});
db.ActorAddress.createIndex({"DelegatedAddress":1}, {"sparse": true});
db.ActorAddress.createIndex({"Epoch":1}, {"sparse": true});

db.ActorBalance.createIndex({"Addresses":1}, {"sparse": true});
db.ActorBalance.createIndex({"Code":1}, {"sparse": true});
db.ActorBalance.createIndex({"Addresses":1,"Code":1}, {"sparse": true});

db.ActorEvent.createIndex({"ActorID":1,"Epoch":1,"LogIndex":1}, {"sparse": true});

db.ActorMessage.createIndex({"ActorID":1,"IsBlock":1,"Epoch":1}, {"sparse": true});
db.ActorMessage.createIndex({"ActorID":1,"IsBlock":1,"MethodName":1,"Epoch":1}, {"sparse": true});
db.ActorMessage.createIndex({"ActorID":1,"ExitCode":1,"Type":1,"Epoch":1,"Value":1}, {"sparse": true});

db.ActorState.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.AllocatedSectors.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Allocations.createIndex({"Client":1}, {"sparse": true});
db.Allocations.createIndex({"Provider":1}, {"sparse": true});
db.Allocations.createIndex({"Data":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Client":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Provider":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Client":1,"AllocationID":1}, {"sparse": true});
db.Allocations.createIndex({"Epoch":1,"Provider":1,"AllocationID":1}, {"sparse": true});

db.BlockHeader.createIndex({"Epoch":1,"Miner":1}, {"sparse": true});

db.BlockMessage.createIndex({"Messages":1}, {"sparse": true});

db.ChangedActor.createIndex({"ActorID":1}, {"sparse": true});
db.ChangedActor.createIndex({"Code":1}, {"sparse": true});
db.ChangedActor.createIndex({"Epoch":1}, {"sparse": true});
db.ChangedActor.createIndex({"ActorID":1,"Epoch":1}, {"sparse": true});

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

db.DealProposal.createIndex({"VerifiedDeal":1,"_id":1}, {"sparse": true});
db.DealProposal.createIndex({"ProviderID":1,"_id":-1}, {"sparse": true});
db.DealProposal.createIndex({"ClientID":1,"_id":-1}, {"sparse": true});
db.DealProposal.createIndex({"ProviderID":1,"VerifiedDeal":1,"_id":-1}, {"sparse": true});
db.DealProposal.createIndex({"ClientID":1,"VerifiedDeal":1,"_id":-1}, {"sparse": true});
db.DealProposal.createIndex({"Epoch":1}, {"sparse": true});
db.DealProposal.createIndex({"Epoch":1,"_id":1}, {"sparse": true});

db.DealProposalDetail.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.DealProposalSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.EthHash.createIndex({"Cid":1}, {"sparse": true});

// no indexes for EventsRoot

db.EvmInitCode.createIndex({"Epoch":1}, {"sparse": true});

db.ExecGas.createIndex({"Epoch":1}, {"sparse": true});

db.ExecTrace.createIndex({"Cid":1}, {"sparse": true});
db.ExecTrace.createIndex({"SignedCid":1}, {"sparse": true});
db.ExecTrace.createIndex({"Epoch":1}, {"sparse": true});
db.ExecTrace.createIndex({"IsBlock":1,"MsgRct.ExitCode":1,"Msg.Method":1,"Epoch":1}, {"sparse": true});
db.ExecTrace.createIndex({"IsBlock":1,"Epoch":1}, {"sparse": true});
db.ExecTrace.createIndex({"IsBlock":1,"Msg.MethodName":1,"Epoch":1}, {"sparse": true});

db.ExplicitMessage.createIndex({"Epoch":1}, {"sparse": true});
db.ExplicitMessage.createIndex({"MethodName":1,"Epoch":1}, {"sparse": true});
db.ExplicitMessage.createIndex({"ExitCode":1,"Epoch":1,"Value":1}, {"sparse": true});

// no indexes for FilSupply

db.FinalHeight.createIndex({"Cids":1}, {"sparse": true});

db.MarketFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.Message.createIndex({"SignedCid":1}, {"sparse": true});

db.MinerDealSector.createIndex({"Epoch":1,"Miner":1}, {"sparse": true});

db.MinerFunds.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MinerSector.createIndex({"Miner":1,"SectorNumber":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"SimpleQaPower":1,"SectorNumber":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"VerifiedDealWeight":1,"SectorNumber":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"DealWeight":1,"SectorNumber":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"DealIDs":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"Expiration":1}, {"sparse": true});
db.MinerSector.createIndex({"Epoch":1,"Activation":1}, {"sparse": true});
db.MinerSector.createIndex({"Miner":1,"Terminated":1,"SectorNumber":1}, {"sparse": true});

db.MinerSectorSummary.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MiningProfitability.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.MultisigBalance.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

db.PendingTxns.createIndex({"Addr":1}, {"sparse": true});
db.PendingTxns.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});
db.PendingTxns.createIndex({"Epoch":1,"Addr":1,"Detail.TxnID":1}, {"sparse": true});

db.SectorClaim.createIndex({"Provider":1}, {"sparse": true});
db.SectorClaim.createIndex({"Provider":1,"Sector":1}, {"sparse": true});

db.StateFinalHeight.createIndex({"Cids":1}, {"sparse": true});

db.Tipset.createIndex({"ChildEpoch":1}, {"sparse": true});

db.VerifiedRegistry.createIndex({"Epoch":1,"Addr":1}, {"sparse": true});

