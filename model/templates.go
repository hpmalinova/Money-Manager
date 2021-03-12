package model

type PayTemplate struct {
	Balance    int
	Categories []Category
	Friends    []string
}

type DLTemplate struct {
	StatusID    int
	Amount      int
	Description string
}

type DebtsTemplate struct {
	Active  []DebtTemplate
	Pending []DebtTemplate
	Balance int
}

type DebtTemplate struct {
	Creditor string
	DLTemplate
}

type LoansTemplate struct {
	Active  []LoanTemplate
	Pending []LoanTemplate
	Balance int
}

type LoanTemplate struct {
	Debtor string
	DLTemplate
}
