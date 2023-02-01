package model

type TransferMessagesForLargeAmountRes struct {
	TotalCount                     int64             `json:"totalCount"`
	TransferMessagesForLargeAmount []TransferMessage `json:"transferMessagesForLargeAmount"`
}
