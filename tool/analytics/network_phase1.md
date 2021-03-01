# Network Metrics Phase1
extracting basic data from chain for the targets below

## core network system health
most of which can be observed in power.State

- [x] Network RB power
- [x] Network QA power
- [x] Network QA power - position estimate
- [x] Network QA power - velocity estimate
- [x] Per Epoch Reward - actual
- [x] Per Epoch Reward - position estimate
- [x] Per Epoch Reward - velocity estimate
- [] Upcoming Sector Expiration - should be extracted regularly from miner.State

## token circulating supply
- [x] Genesis Vesting Rate - sum of multisig.State.AmountUnlocked
- [x] Minting Rate from Storage Mining - from reward.State
- [x] Reward Vesting Rate - from miner.State
- [x] Initial Pledge Rate - sum of miner.State.InitialPledge
- [x] Locking Rate - from power.State, miner.State, market.State
- [x] Burn Rate - from burn addr's balance
- [x] Deal Collateral Locking Rate
- [] Upcoming Initial Pledge Unlock Rate - with deal stats

## mining profitability
- [x] Initial Pledge per 32GiB QA Power
- [x] Initial Consensus Pledge per 32GiB QA power
- [x] Initial Storage Pledge per 32GiB QA power
- [x] Projection of the initial pledge per unit per 32 GiB QA power
- [x] Projection of the fault fee per unit of 32 QA Power
- [] Cost of token production
- [] Token price

## storage reliability
- [] Rate of missing WindowPoSt
- [] Fault fee per epoch
- [] DeclareFault count per epoch
- [] DeclareFaultsRecovered per epoch
- [] Sector lifetime distribution
- [] Sector lifetime distribution by miner
- [] SectorExtension per epoch
- [] SectorTermination per epoch
- [] Regular deal termination per epoch
- [] Filecoin Plus deal termination per epoch
- [] Termination fee per epoch

## gas monitoring
- [x] BlockGasLimit per epoch
- [x] BlockGasUsage per epoch
- [x] BaseFee per epoch
- [x] GasPremium per epoch
- [x] Miner penalty per epoch
- [x] BaseFee burn per epoch
- [x] Overestimate burn per epoch
- [x] Number of messages by message type per epoch
- [x] GasUsage by message type per epoch

## storage demand & filecoin plus
- [] Number of regular deals
- [] Number of Filecoin Plus deals
- [x] Amount of DataCap allocated to notaries
- [x] Amount of DataCap allocated from notaries to clients
- [] Flow of DataCap allocation from each notaries
- [] Flow of DataCap allocation from each Filecoin Plus Clients
