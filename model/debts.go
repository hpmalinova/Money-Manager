package model

//CREATE TABLE debt_status (
//id INT AUTO_INCREMENT PRIMARY KEY,
//status enum('ongoing','pending') NOT NULL,
//amount INT NOT NULL
//);
//
//-- todo foreign key? - creditor/debtor/category_id/status_id
//-- todo delete status when debt is payed
//CREATE TABLE debts (
//creditor INT NOT NULL,
//debtor INT NOT NULL,
//amount INT NOT NULL,
//category_id INT NOT NULL,
//description  VARCHAR (128),
//status_id INT NOT NULL
//);

//`json:"id" validate:"numeric,gte=0"`
//Username string `json:"username" validate:"required,min=3,max=32"`
// TODO JSON
type DebtAndLoan struct {
	CreditorID  int
	DebtorID    int
	Amount      int
	CategoryID  int
	Description string
	StatusID    int
}

// I've given 50lv to Peter for "Happy"
type Loan struct {
	DebtorID    int
	Amount      int
	Description string
}

// I've taken 50lv from George for "Happy"
type Debt struct {
	CreditorID  int
	Amount      int
	Description string
}

//// Peter tries to repay me 20lv for "Happy"
//type PendingLoan struct {
//	DebtorID int
//	Amount int
//
//}
//
//type PendingDebts struct {
//
//}
