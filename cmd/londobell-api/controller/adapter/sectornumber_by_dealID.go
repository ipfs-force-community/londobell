package adapter

import (
	"context"
	"fmt"
	"net/http"

	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	actorstypes "github.com/filecoin-project/go-state-types/actors"
	miner8 "github.com/filecoin-project/go-state-types/builtin/v8/miner"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	miner4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	builtin6 "github.com/filecoin-project/specs-actors/v6/actors/builtin"
	miner6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
	builtin7 "github.com/filecoin-project/specs-actors/v7/actors/builtin"
	miner7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"

	miner10 "github.com/filecoin-project/go-state-types/builtin/v10/miner"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gin-gonic/gin"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
	"github.com/ipfs-force-community/londobell/cmd/londobell-api/util"
)

func GetSectorNumberByDealID(c *gin.Context) {
	alog := log.With("method", "GetSectorNumberByDealID")
	req := model.SectorNumberByDealIDReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := API.GetAppropriateAPI()

	stor := store.ActorStore(context.TODO(), blockstore.NewAPIBlockstore(api))
	addr, err := address.NewFromString(req.Miner)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	act, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	resData := model.SectorNumberByDealIDRes{
		Miner:  req.Miner,
		DealID: abi.DealID(req.DealID),
	}

	err = getSectorByDealIDResByCode(ctx, act, stor, req.DealID, &resData)
	if err != nil {
		alog.Error(err)
		util.ReturnOnErr(c, err)
		return
	}

	res.Data = resData
	c.JSON(http.StatusOK, res)
}

func getSectorByDealIDResByCode(ctx context.Context, mact *types.Actor, stor adt.Store, dealID uint64, resData *model.SectorNumberByDealIDRes) (err error) {
	if name, av, ok := actors.GetActorMetaByCode(mact.Code); ok {
		if name != manifest.MinerKey {
			return fmt.Errorf("actor code is not miner: %s", name)
		}

		switch av {

		case actorstypes.Version8:
			state := miner8.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				return err
			}

			sectors, err := miner8.LoadSectors(stor, state.Sectors)
			if err != nil {
				return fmt.Errorf("load sectors from adt store: %w", err)
			}

			var out miner8.SectorOnChainInfo
			err = sectors.ForEach(&out, func(n int64) error {
				for _, id := range out.DealIDs {
					if id == abi.DealID(dealID) {
						resData.SectorNumber = out.SectorNumber
						return nil
					}
				}

				return nil
			})

			if err != nil {
				return err
			}

			return nil
		case actorstypes.Version9:
			state := miner9.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				return err
			}

			sectors, err := miner9.LoadSectors(stor, state.Sectors)
			if err != nil {
				return fmt.Errorf("load sectors from adt store: %w", err)
			}

			var out miner9.SectorOnChainInfo
			err = sectors.ForEach(&out, func(n int64) error {
				for _, id := range out.DealIDs {
					if id == abi.DealID(dealID) {
						resData.SectorNumber = out.SectorNumber
						return nil
					}
				}

				return nil
			})

			if err != nil {
				return err
			}

			return nil

			// v10
		case actorstypes.Version10:
			state := miner10.State{}
			err = stor.Get(ctx, mact.Head, &state)
			if err != nil {
				return err
			}

			sectors, err := miner10.LoadSectors(stor, state.Sectors)
			if err != nil {
				return fmt.Errorf("load sectors from adt store: %w", err)
			}

			var out miner10.SectorOnChainInfo
			err = sectors.ForEach(&out, func(n int64) error {
				for _, id := range out.DealIDs {
					if id == abi.DealID(dealID) {
						resData.SectorNumber = out.SectorNumber
						return nil
					}
				}

				return nil
			})

			if err != nil {
				return err
			}

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

		sectors, err := miner0.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner0.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil
	case builtin2.StorageMinerActorCodeID:
		state := miner2.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner2.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner2.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	case builtin3.StorageMinerActorCodeID:
		state := miner3.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner3.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner3.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	case builtin4.StorageMinerActorCodeID:
		state := miner4.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner4.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner4.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	case builtin5.StorageMinerActorCodeID:
		state := miner5.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner5.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner5.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	case builtin6.StorageMinerActorCodeID:
		state := miner6.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner6.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner6.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	case builtin7.StorageMinerActorCodeID:
		state := miner7.State{}
		err = stor.Get(ctx, mact.Head, &state)
		if err != nil {
			return err
		}

		sectors, err := miner7.LoadSectors(stor, state.Sectors)
		if err != nil {
			return fmt.Errorf("load sectors from adt store: %w", err)
		}

		var out miner7.SectorOnChainInfo
		err = sectors.ForEach(&out, func(n int64) error {
			for _, id := range out.DealIDs {
				if id == abi.DealID(dealID) {
					resData.SectorNumber = out.SectorNumber
					return nil
				}
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil

	}

	return fmt.Errorf("unknown actor code %s", mact.Code)
}
