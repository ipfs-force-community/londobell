package account

type PeriodBillRes struct {
	Income  string
	Pay     string
	GasCost string
}

type PeriodIncome struct {
	TotalValue string
}

type PeriodPay struct {
	TotalValue string
}

type PeriodGasCost struct {
	TotalGasCost string
}
