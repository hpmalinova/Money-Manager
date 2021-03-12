package model

type Pay struct {
	UserID       int    `json:"userID" validate:"numeric,gte=0"` // TODO is needed?
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
}

type Transfer struct {
	CreditorID int    `json:"debtorID" validate:"numeric,gte=0"`
	LoanID     int    `json:"loanID" validate:"numeric,gte=0"`
	DebtID     int    `json:"debtID" validate:"numeric,gte=0"`
	DebtName   string `json:"debtName" validate:"required,min=3,max=32"`
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
