package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	adt0 "github.com/filecoin-project/specs-actors/actors/util/adt"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	adt4 "github.com/filecoin-project/specs-actors/v4/actors/util/adt"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	adt5 "github.com/filecoin-project/specs-actors/v5/actors/util/adt"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	adt6 "github.com/filecoin-project/specs-actors/v6/actors/util/adt"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	adt7 "github.com/filecoin-project/specs-actors/v7/actors/util/adt"
	builtin8 "github.com/filecoin-project/specs-actors/v8/actors/builtin"
	miner8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/miner"
	adt8 "github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetBatchMinersInfo(c *gin.Context) {
	req := model.BatchMinersReq{}
	res := model.CommonRes{Code: model.Success}
	batchRes := model.BatchMinersRes{}
	err := c.BindJSON(&req)
	if err != nil {
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	fmt.Printf("BatchMinersReq: %v\n", req)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	if req.Epoch == 0 {
		ts, err = API.ChainHead(ctx)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	for _, Miner := range req.Miners {
		maddr, err := address.NewFromString(Miner.Miner)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		mi, err := API.StateMinerInfo(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		power, err := API.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		mact, err := API.StateGetActor(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		if !builtin.IsStorageMinerActor(mact.Code) {
			res.Code = model.Fail
			res.Msg = "provided address does not correspond to a miner actor"
			c.JSON(http.StatusOK, res)
			return
		}

		availableBalance, err := API.StateMinerAvailableBalance(ctx, maddr, ts.Key())
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(API))
		mas, err := miner.Load(stor, mact)
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		lockedFunds, err := mas.LockedFunds()
		if err != nil {
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		var (
			sectorCount          = uint64(0)
			faultSectorCount     = uint64(0)
			activeSectorCount    = uint64(0)
			liveSectorCount      = uint64(0)
			recoverSectorCount   = uint64(0)
			terminateSectorCount = uint64(0)
			precommitSectorCount = uint64(0)
		)

		resData := model.MinerRes{}

		switch mact.Code {
		case builtin0.StorageMinerActorCodeID:
			state := miner0.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner0.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner0.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt0.AsMap(stor, state.PreCommittedSectors)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner0.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin2.StorageMinerActorCodeID:
			state := miner2.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner2.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner2.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})

			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt2.AsMap(stor, state.PreCommittedSectors)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner2.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin3.StorageMinerActorCodeID:
			state := miner3.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner3.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner3.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt3.AsMap(stor, state.PreCommittedSectors, builtin3.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner3.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin4.StorageMinerActorCodeID:
			state := miner4.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner4.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner4.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt4.AsMap(stor, state.PreCommittedSectors, builtin4.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner4.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin5.StorageMinerActorCodeID:
			state := miner5.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner5.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner5.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt5.AsMap(stor, state.PreCommittedSectors, builtin5.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}
			var precommit miner5.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin6.StorageMinerActorCodeID:
			state := miner6.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner6.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner6.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt6.AsMap(stor, state.PreCommittedSectors, builtin6.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner6.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin7.StorageMinerActorCodeID:
			state := miner7.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner7.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner7.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt7.AsMap(stor, state.PreCommittedSectors, builtin7.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner7.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		case builtin8.StorageMinerActorCodeID:
			state := miner8.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner8.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner8.Partition
				return partitions.ForEach(&part, func(partIdx int64) error {
					sc, err := part.Sectors.Count()
					if err != nil {
						return err
					}
					sectorCount += sc

					fc, err := part.Faults.Count()
					if err != nil {
						return err
					}
					faultSectorCount += fc

					active, err := part.ActiveSectors()
					if err != nil {
						return err
					}
					ac, err := active.Count()
					if err != nil {
						return err
					}
					activeSectorCount += ac

					live, err := part.LiveSectors()
					if err != nil {
						return err
					}
					lc, err := live.Count()
					if err != nil {
						return err
					}
					liveSectorCount += lc

					rc, err := part.Recoveries.Count()
					if err != nil {
						return err
					}
					recoverSectorCount += rc

					tc, err := part.Terminated.Count()
					if err != nil {
						return err
					}
					terminateSectorCount += tc

					return nil
				})
			})
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			precommitted, err := adt8.AsMap(stor, state.PreCommittedSectors, builtin8.DefaultHamtBitwidth)
			if err != nil {
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			var precommit miner8.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
		}

		resData.Epoch = ts.Height()
		resData.Miner = maddr
		resData.Owner = mi.Owner
		resData.Worker = mi.Worker
		resData.Controllers = mi.ControlAddresses
		resData.SectorSize = mi.SectorSize
		resData.Power = power.MinerPower.RawBytePower
		resData.QualityPower = power.MinerPower.QualityAdjPower
		resData.Balance = mact.Balance
		resData.AvailableBalance = availableBalance
		resData.VestingFunds = lockedFunds.VestingFunds
		resData.LockedFunds = lockedFunds.PreCommitDeposits
		resData.InitialPledgeRequirement = lockedFunds.InitialPledgeRequirement
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		batchRes.MinersRes = append(batchRes.MinersRes, resData)
	}

	res.Data = batchRes
	c.JSON(http.StatusOK, res)
}
