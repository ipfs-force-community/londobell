package miner

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ipfs-force-community/londobell/cmd/londobell-api/model"
)

type SectorHealthRes struct {
	SectorHealths []SectorHealth
}

type SectorHealth struct {
	Epoch                 int64
	FaultSectors          int64
	RecoveriesSectors     int64
	UnprovenSectors       int64
	ActiveSectors         int64
	LiveSectors           int64
	AllSectors            int64
	ActiveSectorsQAPower  string
	FaultsQAPower         string
	RecoveriesQAPower     string
	UnprovenQAPower       string
	ActiveSectorsRawPower string
	FaultsRawPower        string
	RecoveriesRawPower    string
	UnprovenRawPower      string
	TerminatedSectors     int64
}

func NewMockSectorHealthRes() SectorHealthRes {
	return SectorHealthRes{
		[]SectorHealth{
			{
				Epoch:                 123,
				FaultSectors:          123,
				RecoveriesSectors:     123,
				UnprovenSectors:       123,
				ActiveSectors:         123,
				LiveSectors:           123,
				AllSectors:            123,
				ActiveSectorsQAPower:  "123",
				FaultsQAPower:         "123",
				RecoveriesQAPower:     "123",
				UnprovenQAPower:       "123",
				ActiveSectorsRawPower: "123",
				FaultsRawPower:        "123",
				RecoveriesRawPower:    "123",
				UnprovenRawPower:      "123",
				TerminatedSectors:     123,
			},
			{
				Epoch:                 234,
				FaultSectors:          234,
				RecoveriesSectors:     234,
				UnprovenSectors:       234,
				ActiveSectors:         234,
				LiveSectors:           234,
				AllSectors:            234,
				ActiveSectorsQAPower:  "234",
				FaultsQAPower:         "234",
				RecoveriesQAPower:     "234",
				UnprovenQAPower:       "234",
				ActiveSectorsRawPower: "234",
				FaultsRawPower:        "234",
				RecoveriesRawPower:    "234",
				UnprovenRawPower:      "234",
				TerminatedSectors:     234,
			},
		},
	}
}

func (m SectorHealthRes) GetMockResponse(c *gin.Context) {
	res := model.CommonRes{Code: model.Success}
	res.Data = m

	c.JSON(http.StatusOK, res)
}
