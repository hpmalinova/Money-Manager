package model

type PayTemplate struct {
	Balance    int
	Categories []Category
	Friends    []string
}

type DebtsTemplate struct {
	Active  []DebtTemplate
	Pending []DebtTemplate
	Balance int
}

type DebtTemplate struct {
	StatusID    int
	Creditor    string
	Amount      int
	Description string
}
