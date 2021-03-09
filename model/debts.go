package model

type DebtAndLoan struct {
	CreditorID   int    `json:"creditorID" validate:"numeric,gte=0"`
	DebtorID     int    `json:"debtorID" validate:"numeric,gte=0"`
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
	StatusID     int    `json:"statusID" validate:"numeric,gte=0"`
}

// I've given 50lv to Peter for "Happy"
type Loan struct {
	DebtorID    int    `json:"debtorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

// I've taken 50lv from George for "Happy"
type Debt struct {
	CreditorID  int    `json:"creditorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}
