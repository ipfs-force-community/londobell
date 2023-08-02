package model

import (
	"github.com/shopspring/decimal"
)

type QAPowerRes struct {
	VDCPower decimal.Decimal
	DCPower  decimal.Decimal
	CCPower  decimal.Decimal
}
