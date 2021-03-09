package model

type Pay struct {
	UserID       int    `json:"userID" validate:"numeric,gte=0"` // TODO is needed?
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
}

type Transfer struct {
	CreditorID int `json:"debtorID" validate:"numeric,gte=0"`
	LoanID     int `json:"loanID" validate:"numeric,gte=0"`
	DebtID     int `json:"debtID" validate:"numeric,gte=0"`
	GiveTo
}

type GiveTo struct {
	DebtorID    int    `json:"debtorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type Give struct {
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	GiveTo
}

type RepayRequest struct {
	DebtID int `json:"debtID" validate:"numeric,gte=0"`
	Amount int `json:"amount" validate:"numeric,gte=0"`
}

type Split struct {
	CreditorID int `json:"debtorID" validate:"numeric,gte=0"`
	Give
}