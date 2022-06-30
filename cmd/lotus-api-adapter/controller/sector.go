package controller

import (
	"context"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
	builtin8 "github.com/filecoin-project/specs-actors/v8/actors/builtin"
	miner8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/miner"
	"github.com/gin-gonic/gin"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetSectorInfo(c *gin.Context) {
	req := model.SectorReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetSectorInfo] bind json SectorReq err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ts *types.TipSet
	if req.Epoch == 0 {
		ts, err = API.ChainHead(ctx)
		if err != nil {
			log.Errorf("[GetSectorInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetSectorInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		log.Errorf("[GetSectorInfo] address.NewFromString err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	mact, err := API.StateGetActor(ctx, maddr, ts.Key())
	if err != nil {
		log.Errorf("[GetSectorInfo] api StateGetActor err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(API))

	resDatas := make([]model.SectorRes, 0)

	switch mact.Code {
	case builtin0.StorageMinerActorCodeID:
		state := miner0.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner0.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner0.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner0 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = 0
			resData.ReplaceDayReward = big.NewInt(0)

			resDatas = append(resDatas, resData)
		})
	case builtin2.StorageMinerActorCodeID:
		state := miner2.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner2.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner2.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner2 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin3.StorageMinerActorCodeID:
		state := miner3.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner3.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner3.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner3 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin4.StorageMinerActorCodeID:
		state := miner4.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner4.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner4.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner4 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin5.StorageMinerActorCodeID:
		state := miner5.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner5.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner5.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner5 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin6.StorageMinerActorCodeID:
		state := miner6.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner6.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner6.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner6 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin7.StorageMinerActorCodeID:
		state := miner7.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner7.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner7.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner7 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	case builtin8.StorageMinerActorCodeID:
		state := miner8.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			log.Errorf("[GetSectorInfo] stor.Get miner8.State err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		err = state.ForEachSector(stor, func(info *miner8.SectorOnChainInfo) {
			resData := model.SectorRes{}
			resData.Miner = maddr
			resData.Date = CalcTimeByEpoch(uint64(info.Expiration))
			resData.SectorNumber = info.SectorNumber

			if info.SealProof >= 0 && info.SealProof <= 4 {
				resData.Version = "V1"
			} else {
				resData.Version = "V1_1"
			}

			resData.Size, err = info.SealProof.SectorSize()
			if err != nil {
				log.Errorf("[GetSectorInfo] miner8 SectorSize err: %w", err)
				res.Code = model.Fail
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			resData.Activation = info.Activation
			resData.Expiration = info.Expiration
			resData.Pledge = info.InitialPledge
			resData.DealWeight = info.DealWeight
			resData.VerifiedDealWeight = info.VerifiedDealWeight
			resData.ExpectedDayReward = info.ExpectedDayReward
			resData.ExpectedStoragePledge = info.ExpectedStoragePledge
			resData.ReplaceSectorAge = info.ReplacedSectorAge
			resData.ReplaceDayReward = info.ReplacedDayReward

			resDatas = append(resDatas, resData)
		})
	}
	if err != nil {
		log.Errorf("[GetSectorInfo] state.ForEachSector err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	res.Data = resDatas
	c.JSON(http.StatusOK, res)
}
