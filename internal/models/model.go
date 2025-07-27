package models

type TransferRequest struct {
	SenderUserID   int `json:"senderUserId"`
	ReceiverUserID int `json:"receiverUserId"`
	Amount         int `json:"amount"`
}

type SingleUserRequest struct {
	UserID      int `json:"userId"`
	UserBalance int `json:"userBalance"`
}
