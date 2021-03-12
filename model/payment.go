package model

type Pay struct {
	UserID       int    `json:"userID" validate:"numeric,gte=0"` // TODO is needed?
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
}

type TransferLoan struct {
	DebtCategoryID    int    `json:"debtCategoryID" validate:"numeric,gte=0"`
	RepayCategoryName string `json:"repayCategoryName" validate:"required,min=3,max=32"`
	Transfer
}

type TransferSplit struct {
	Expense Category
	Transfer
}

type Transfer struct {
	CreditorID     int `json:"debtorID" validate:"numeric,gte=0"`
	LoanCategoryID int `json:"loanCategoryID" validate:"numeric,gte=0"`
	Loan
}

type Debt struct {
	CreditorID  int    `json:"creditorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type DebtExt struct {
	StatusID     int    `json:"statusID" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Debt
}

type Loan struct {
	DebtorID    int    `json:"debtorID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type LoanExt struct {
	StatusID int `json:"statusID" validate:"numeric,gte=0"`
	Loan
}

type Give struct {
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Loan
}

type RepayRequest struct {
	DebtID int `json:"debtID" validate:"numeric,gte=0"`
	Amount int `json:"amount" validate:"numeric,gte=0"`
}

type Split struct {
	CreditorID int `json:"debtorID" validate:"numeric,gte=0"`
	Give
}

type History struct {
	UserID      int    `json:"userID" validate:"numeric,gte=0"`
	Amount      int    `json:"amount" validate:"numeric,gte=0"`
	CategoryID  int    `json:"categoryID" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
}

type HistoryShow struct {
	Amount       int
	CategoryName string
	CategoryType string
	Description  string
}

type HistoryShowAll struct {
	HistoryShowAll []HistoryShow
}

type Statistics struct {
	Ratios []Ratio
}

type Ratio struct {
	Percent      string
	CategoryName string
}

type Accept struct {
	StatusID int      `json:"statusID" validate:"numeric,gte=0"`
	RepayC   Category `json:"repayC"`
	ExpenseC Category `json:"expenseC"`
}

type AcceptPayment struct {
	CreditorID  int    `json:"creditorID" validate:"numeric,gte=0"`
	DebtorID    int    `json:"debtorID" validate:"numeric,gte=0"`
	DebtAmount  int    `json:"debtAmount" validate:"numeric,gte=0"`
	Description string `json:"description,omitempty"`
	Status
}

type Status struct {
	StatusID      int    `json:"statusID" validate:"numeric,gte=0"`
	Status        string `json:"status"`
	PendingAmount int    `json:"pendingAmount" validate:"numeric,gte=0"`
}
