package models

type EnrollRequest struct {
	Id     int `json:"id"`
	Amount int `json:"amount"`
}

type WithdrawRequest struct {
	Id     int `json:"id"`
	Amount int `json:"amount"`
}

type TransferRequest struct {
	WithdrawId int `json:"withdrawId"`
	EnrollId   int `json:"enrollId"`
	Amount     int `json:"amount"`
}

type BalanceRequest struct {
	Id int `json:"id"`
}
