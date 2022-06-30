package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/urfave/cli/v2"

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
