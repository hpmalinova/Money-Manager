package model

type History struct {
	UserID      int    `json:"userID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	CategoryID  int    `json:"categoryID" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type LoanHistory struct {
	DebtorID int `json:"debtorID" validate:"numeric,gte=0"`
	History
}

type DebtHistory struct {
	CreditorID int `json:"creditorID" validate:"numeric,gte=0"`
	History
}
