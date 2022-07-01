package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lotus/lib/tablewriter"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/model"
)

func GetSectorPower(cctx *cli.Context) error {
	mAddr := cctx.String("miner")
	sector := cctx.Uint64("sector")

	var params = map[string]interface{}{
		"miner": mAddr,
		"epoch": cctx.Int64("epoch"),
	}
	var reader io.Reader

	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	reader = bytes.NewBuffer(body)

	url := "http://106.14.10.70:12345/sector"
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	var client = http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("resp.StatusCode != 200")
	}

	defer resp.Body.Close()

	var result model.CommonRes

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	resByte, err := json.Marshal(result.Data)
	if err != nil {
		return err
	}

	var sectorInfos []model.SectorRes
	err = json.Unmarshal(resByte, &sectorInfos)
	if err != nil {
		return err
	}

	var (
		size               abi.SectorSize
		activation         abi.ChainEpoch
		expiration         abi.ChainEpoch
		dealWeight         abi.DealWeight
		verifiedDealWeight abi.DealWeight
		duration           abi.ChainEpoch
	)

	for _, sectorInfo := range sectorInfos {
		if sectorInfo.SectorNumber == abi.SectorNumber(sector) {
			size = sectorInfo.Size
			activation = sectorInfo.Activation
			expiration = sectorInfo.Expiration
			dealWeight = sectorInfo.DealWeight
			verifiedDealWeight = sectorInfo.VerifiedDealWeight
			duration = expiration - activation
			break
		}
	}

	power := miner.QAPowerForWeight(size, duration, dealWeight, verifiedDealWeight)

	w := tablewriter.New(tablewriter.Col("miner"),
		tablewriter.Col("sector"),
		tablewriter.Col("power"))
	w.Write(map[string]interface{}{
		"miner":  mAddr,
		"sector": sector,
		"power":  power})

	return w.Flush(os.Stdout)
}

func GetSectorPowerInfo(c *gin.Context) {
	req := model.SectorPowerReq{}
	res := model.CommonRes{Code: model.Success}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] bind json SectorPowerReq err: %w", err)
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
			log.Errorf("[GetSectorPowerInfo] api ChainHead err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		ts, err = API.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(req.Epoch), types.EmptyTSK)
		if err != nil {
			log.Errorf("[GetSectorPowerInfo] api ChainGetTipSetByHeight err: %w", err)
			res.Code = model.Fail
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}

	maddr, err := address.NewFromString(req.Miner)
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] address.NewFromString err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	resData := model.SectorPowerRes{}

	si, err := API.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(req.Sector), ts.Key())
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] API.StateSectorGetInfo err: %w", err)
	}

	resData.Miner = maddr
	resData.Date = CalcTimeByEpoch(uint64(si.Expiration))
	resData.SectorNumber = si.SectorNumber
	if si.SealProof >= 0 && si.SealProof <= 4 {
		resData.Version = "V1"
	} else {
		resData.Version = "V1_1"
	}
	resData.Size, err = si.SealProof.SectorSize()
	if err != nil {
		log.Errorf("[GetSectorPowerInfo] SectorSize err: %w", err)
		res.Code = model.Fail
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	resData.Activation = si.Activation
	resData.Expiration = si.Expiration
	resData.Pledge = si.InitialPledge
	resData.DealWeight = si.DealWeight
	resData.VerifiedDealWeight = si.VerifiedDealWeight
	resData.ExpectedDayReward = si.ExpectedDayReward
	resData.ExpectedStoragePledge = si.ExpectedStoragePledge
	resData.ReplaceSectorAge = si.ReplacedSectorAge
	resData.ReplaceDayReward = si.ReplacedDayReward

	res.Data = resData
	c.JSON(http.StatusOK, res)
}
