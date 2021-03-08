package repository

import (
	"database/sql"
	"fmt"
	"github.com/hpmalinova/Money-Manager/model"
	"log"
	"strconv"
)

type HistoryRepoMysql struct {
	db *sql.DB
}

// TODO remove money from the user wallet!

func NewHistoryRepoMysql(user, password, dbname string) *HistoryRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &HistoryRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	return repo
}

func (h *HistoryRepoMysql) Pay(history *model.History) error {
	statement := "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err := h.db.Exec(statement, history.UserID, history.Amount, history.CategoryID, history.Description)
	if err != nil {
		return err
	}

	return nil
}

// I gave 50lv to George for "Happy"
// Category: Expense - LOAN
func (h *HistoryRepoMysql) GiveLoan(loan *model.LoanHistory) error {
	description := "Loan for" + strconv.Itoa(loan.DebtorID) + loan.Description

	return h.moneyTransfer(&loan.History, description)
}

// George repaid 50lv for "Happy"
// Category: Income - REPAY
func (h *HistoryRepoMysql) ReceiveDebt(loan *model.LoanHistory) error {
	description := "Repay from" + strconv.Itoa(loan.DebtorID) + loan.Description
	//statement := "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	//_, err := h.db.Exec(statement, loan.UserID, loan.Amount, loan.CategoryID, description)
	//if err != nil {
	//	return err
	//}

	return h.moneyTransfer(&loan.History, description)
}

// George paid 50lv for "Happy" with Peter's money
// Category: Expense - FOOD
func (h *HistoryRepoMysql) PayWithDebt(debt *model.DebtHistory) error {
	description := "Debt from" + strconv.Itoa(debt.CreditorID) + debt.Description

	return h.moneyTransfer(&debt.History, description)
}

func (h *HistoryRepoMysql) moneyTransfer(history *model.History, description string) error {
	statement := "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err := h.db.Exec(statement, history.UserID, history.Amount, history.CategoryID, description)
	if err != nil {
		return err
	}

	return nil
}
