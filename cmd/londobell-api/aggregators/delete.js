// 325155,410990
// 410991,444958
// 444959,488159,...
db.ActorBalance.deleteMany({"Epoch":{$gt:488156}});

db.ActorState.deleteMany({"Epoch":{$gt:488156}})

db.BlockHeader.deleteMany({"Epoch":{$gt:488156}})

db.ClaimedPower.deleteMany({"Epoch":{$gt:488156}})

db.DealProposal.deleteMany({"Epoch":{$gt:488156}})

db.DealProposalDetail.deleteMany({"ActorStateExBasic.Epoch":{$gt:488156}})

db.DealProposalSummary.deleteMany({"ActorStateExBasic.Epoch":{$gt:488156}})

db.ExecTrace.deleteMany({"Epoch":{$gt:488156}})

db.FilSupply.deleteMany({"_id":{$gt:488156}})

db.MarketFunds.deleteMany({"Epoch":{$gt:488156}})

db.Message.deleteMany({"Detail.PackedHeight":{$gt:488156}})

db.MinerDealSector.deleteMany({"Epoch":{$gt:488156}})

db.MinerFunds.deleteMany({"Epoch":{$gt:488156}})

db.MinerSectorHealth.deleteMany({"Epoch":{$gt:488156}})

db.MinerSectorSummary.deleteMany({"Epoch":{$gt:488156}})

db.MiningProfitability.deleteMany({"Epoch":{$gt:488156}})

db.MultisigBalance.deleteMany({"Epoch":{$gt:488156}})

db.PendingTxns.deleteMany({"Epoch":{$gt:488156}})

db.Tipset.deleteMany({"_id":{$gt:488156}})

db.VerifiedRegistry.deleteMany({"Epoch":{$gt:488156}})

db.FinalHeight.deleteMany({"_id":{$gt:488156}})
