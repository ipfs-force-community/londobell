// 325155,410990
// 410991,444958
// 444958,488159,...
db.ActorBalance.deleteMany({"Epoch":{$gt:444958}});

db.ActorState.deleteMany({"Epoch":{$gt:444958}})

db.BlockHeader.deleteMany({"Epoch":{$gt:444958}})

db.ClaimedPower.deleteMany({"Epoch":{$gt:444958}})

db.DealProposal.deleteMany({"Epoch":{$gt:444958}})

db.DealProposalDetail.deleteMany({"ActorStateExBasic.Epoch":{$gt:444958}})

db.DealProposalSummary.deleteMany({"ActorStateExBasic.Epoch":{$gt:444958}})

db.ExecTrace.deleteMany({"Epoch":{$gt:444958}})

db.FilSupply.deleteMany({"_id":{$gt:444958}})

db.MarketFunds.deleteMany({"Epoch":{$gt:444958}})

db.Message.deleteMany({"Detail.PackedHeight":{$gt:444958}})

db.MinerDealSector.deleteMany({"Epoch":{$gt:444958}})

db.MinerFunds.deleteMany({"Epoch":{$gt:444958}})

db.MinerSectorHealth.deleteMany({"Epoch":{$gt:444958}})

db.MinerSectorSummary.deleteMany({"Epoch":{$gt:444958}})

db.MiningProfitability.deleteMany({"Epoch":{$gt:444958}})

db.MultisigBalance.deleteMany({"Epoch":{$gt:444958}})

db.PendingTxns.deleteMany({"Epoch":{$gt:444958}})

db.Tipset.deleteMany({"_id":{$gt:444958}})

db.VerifiedRegistry.deleteMany({"Epoch":{$gt:444958}})

db.FinalHeight.deleteMany({"_id":{$gt:444958}})
