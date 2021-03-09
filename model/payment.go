package model

type Pay struct {
	UserID       int    `json:"userID" validate:"numeric,gte=0"` // TODO is needed?
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
}

type GiveLoan struct {
	DebtorID    int    `json:"debtorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type Split struct {
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	GiveLoan
}

type RepayRequest struct {
	DebtID int `json:"debtID" validate:"numeric,gte=0"`
	Amount int `json:"amount" validate:"numeric,gte=0"`
}
