package adapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	miner8 "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	adt8 "github.com/filecoin-project/go-state-types/builtin/v8/util/adt"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
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
	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
)

func GetMinerInfo(c *gin.Context) {
	alog := log.With("method", "GetMinerInfo")
	req := model.MinerReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	api := API.GetAppropriateAPI()
	if req.Epoch == 0 {
		ts, err = api.ChainHead(ctx)
	} else {
		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
	}

	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	mi, err := api.StateMinerInfo(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	power, err := api.StateMinerPower(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	mact, err := api.StateGetActor(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	if !builtin.IsStorageMinerActor(mact.Code) {
		util.ReturnOnErr(c, alog, fmt.Errorf("provided address does not correspond to a miner actor"))
		return
	}

	availableBalance, err := api.StateMinerAvailableBalance(ctx, maddr, ts.Key())
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))
	mas, err := miner.Load(stor, mact)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	lockedFunds, err := mas.LockedFunds()
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	resData := &model.MinerRes{}

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

	err = getMinerResByCode(ctx, mact, stor, resData)
	if err != nil {
		util.ReturnOnErr(c, alog, err)
		return
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}

func getMinerResByCode(ctx context.Context, mact *types.Actor, stor adt.Store, resData *model.MinerRes) (err error) {
	var (
		sectorCount          = uint64(0)
		faultSectorCount     = uint64(0)
		activeSectorCount    = uint64(0)
		liveSectorCount      = uint64(0)
		recoverSectorCount   = uint64(0)
		terminateSectorCount = uint64(0)
		precommitSectorCount = uint64(0)
	)

	if name, av, ok := actors.GetActorMetaByCode(mact.Code); ok {
		if name != actors.MinerKey {
			return fmt.Errorf("actor code is not miner: %s", name)
		}

		switch av {

		case actorstypes.Version8:
			state := miner8.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				return err
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				return err
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
				return err
			}

			precommitted, err := adt8.AsMap(stor, state.PreCommittedSectors, sbuiltin.DefaultHamtBitwidth)
			if err != nil {
				return err
			}

			var precommit miner8.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
			resData.SectorCount = sectorCount
			resData.FaultSectorCount = faultSectorCount
			resData.ActiveSectorCount = activeSectorCount
			resData.LiveSectorCount = liveSectorCount
			resData.RecoverSectorCount = recoverSectorCount
			resData.TerminateSectorCount = terminateSectorCount
			resData.PrecommitSectorCount = precommitSectorCount

			return nil
		case actorstypes.Version9:
			state := miner9.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				return err
			}

			dls, err := state.LoadDeadlines(stor)
			if err != nil {
				return err
			}

			err = dls.ForEach(stor, func(dlIdx uint64, dl *miner9.Deadline) error {
				partitions, err := dl.PartitionsArray(stor)
				if err != nil {
					return err
				}
				var part miner9.Partition
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
				return err
			}

			precommitted, err := adt9.AsMap(stor, state.PreCommittedSectors, sbuiltin.DefaultHamtBitwidth)
			if err != nil {
				return err
			}

			var precommit miner9.SectorPreCommitOnChainInfo
			precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
				precommitSectorCount++
				return nil
			})

			resData.State = state
			resData.SectorCount = sectorCount
			resData.FaultSectorCount = faultSectorCount
			resData.ActiveSectorCount = activeSectorCount
			resData.LiveSectorCount = liveSectorCount
			resData.RecoverSectorCount = recoverSectorCount
			resData.TerminateSectorCount = terminateSectorCount
			resData.PrecommitSectorCount = precommitSectorCount

			return nil
		}
	}

	switch mact.Code {
	case builtin0.StorageMinerActorCodeID:
		state := miner0.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt0.AsMap(stor, state.PreCommittedSectors)
		if err != nil {
			return err
		}

		var precommit miner0.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin2.StorageMinerActorCodeID:
		state := miner2.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt2.AsMap(stor, state.PreCommittedSectors)
		if err != nil {
			return err
		}

		var precommit miner2.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin3.StorageMinerActorCodeID:
		state := miner3.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt3.AsMap(stor, state.PreCommittedSectors, builtin3.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		var precommit miner3.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin4.StorageMinerActorCodeID:
		state := miner4.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt4.AsMap(stor, state.PreCommittedSectors, builtin4.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		var precommit miner4.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin5.StorageMinerActorCodeID:
		state := miner5.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt5.AsMap(stor, state.PreCommittedSectors, builtin5.DefaultHamtBitwidth)
		if err != nil {
			return err
		}
		var precommit miner5.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin6.StorageMinerActorCodeID:
		state := miner6.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt6.AsMap(stor, state.PreCommittedSectors, builtin6.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		var precommit miner6.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	case builtin7.StorageMinerActorCodeID:
		state := miner7.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		dls, err := state.LoadDeadlines(stor)
		if err != nil {
			return err
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
			return err
		}

		precommitted, err := adt7.AsMap(stor, state.PreCommittedSectors, builtin7.DefaultHamtBitwidth)
		if err != nil {
			return err
		}

		var precommit miner7.SectorPreCommitOnChainInfo
		precommitted.ForEach(&precommit, func(string) error { // nolint: errcheck
			precommitSectorCount++
			return nil
		})

		resData.State = state
		resData.SectorCount = sectorCount
		resData.FaultSectorCount = faultSectorCount
		resData.ActiveSectorCount = activeSectorCount
		resData.LiveSectorCount = liveSectorCount
		resData.RecoverSectorCount = recoverSectorCount
		resData.TerminateSectorCount = terminateSectorCount
		resData.PrecommitSectorCount = precommitSectorCount

		return nil
	}

	return fmt.Errorf("unknown actor code %s", mact.Code)
}
