package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/hpmalinova/Money-Manager/model"
	"log"
	"time"
)

type DebtRepoMysql struct {
	db *sql.DB
}

func NewDebtRepoMysql(user, password, dbname string) *DebtRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &DebtRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	repo.db.SetConnMaxLifetime(time.Minute * 5) // todo add to others?
	repo.db.SetMaxOpenConns(10)
	repo.db.SetMaxIdleConns(10)
	repo.db.SetConnMaxIdleTime(time.Minute * 3)

	return repo
}

const (
	ongoingStatus = "ongoing"
	pendingStatus = "pending"
)

func (d *DebtRepoMysql) Add(debt *model.DebtAndLoan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// BEGIN TRANSACTION
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	// DEFER ROLLBACK
	defer tx.Rollback()

	// Add Status
	statement := "INSERT INTO debt_status(status, amount) VALUES(?, ?, ?)"
	result, err := d.db.Exec(statement, ongoingStatus, debt.Amount)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	statusID := int(id)

	// Add Debt
	statement = "INSERT INTO debt(creditor, debtor, amount, category_id, description, status_id) VALUES(?, ?, ?, ?, ?, ?)"
	result, err = d.db.Exec(statement, debt.CreditorID, debt.DebtorID, debt.Amount, debt.CategoryID, debt.Description, statusID)
	if err != nil {
		return err
	}

	numRows, err := result.RowsAffected()
	if err != nil || numRows != 1 {
		msg := fmt.Sprintf("error inserting debt: %s\n", err)
		return errors.New(msg)
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		msg := fmt.Sprintf("error in creating group: %s\n", err)
		return errors.New(msg)
	}
	return nil
}

func (d *DebtRepoMysql) RequestPaymentConfirmation(statusID, amount int) error {
	statement := "SELECT amount FROM debt_status WHERE id = ?"
	var debtAmount int
	err := d.db.QueryRow(statement, statusID).Scan(&debtAmount)
	if err != nil {
		return err
	}
	// TODO
	if amount > debtAmount {
		amount = debtAmount
	}

	statement = "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
	return d.db.QueryRow(statement, pendingStatus, amount, statusID).Scan()
}

// TODO
// accept payment (delete or minimize the debt)
func (d *DebtRepoMysql) AcceptPayment(statusID int) (int, error) {
	statement := "SELECT amount FROM debt_status WHERE id = ?"
	var pendingAmount int
	if err := d.db.QueryRow(statement, statusID).Scan(&pendingAmount); err != nil {
		return 0, err
	}

	statement = "SELECT amount FROM debts WHERE status_id = ?"
	var debtAmount int
	if err := d.db.QueryRow(statement, statusID).Scan(&debtAmount); err != nil {
		return 0, err
	}

	if pendingAmount < debtAmount {
		amountToPay := debtAmount - pendingAmount
		err := d.payPartOfDebt(statusID, amountToPay)
		return 0, errors.New("error in paying part of the debt: " + err.Error())
	} else if pendingAmount == debtAmount {
		err := d.deleteDebt(statusID)
		return 0, errors.New("error in deleting debt: " + err.Error())
	}

	return pendingAmount, nil
}

func (d *DebtRepoMysql) deleteDebt(statusID int) error {
	//user, err := d.FindByID(id) // todo?
	//if err != nil {
	//	return nil, err
	//}
	//statement := fmt.Sprintf("DELETE FROM users WHERE id=%d", id)
	statement := "DELETE FROM debt_status WHERE id=?"
	if _, err := d.db.Exec(statement, statusID); err != nil {
		return err
	}

	statement = "DELETE FROM debts WHERE status_id=?"
	if _, err := d.db.Exec(statement, statusID); err != nil {
		return err
	}

	return nil
}

func (d *DebtRepoMysql) payPartOfDebt(statusID, amountToPay int) error {
	statement := "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
	if err := d.db.QueryRow(statement, ongoingStatus, amountToPay, statusID).Scan(); err != nil {
		return err
	}

	statement = "UPDATE debt SET amount = ? WHERE id = ?"
	return d.db.QueryRow(statement, amountToPay, statusID).Scan()
}

// TODO
// decline payment (change to ongoing)
func (d *DebtRepoMysql) DeclinePayment(statusID int) error {
	statement := "SELECT amount FROM debts WHERE status_id = ?"
	var debtAmount int
	if err := d.db.QueryRow(statement, statusID).Scan(&debtAmount); err != nil {
		return err
	}

	statement = "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
	return d.db.QueryRow(statement, ongoingStatus, debtAmount, statusID).Scan()
}

func (d *DebtRepoMysql) FindActiveLoans(creditorID int) ([]model.Loan, error) {
	// az == creditor & status == ongoing
	statement := `SELECT debtor, amount, description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.creditor = ? AND s.status=?`
	rows, err := d.db.Query(statement, creditorID, ongoingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := []model.Loan{}
	for rows.Next() {
		var loan model.Loan
		err := rows.Scan(&loan.DebtorID, &loan.Amount, &loan.Description)
		if err != nil {
			return nil, err
		}
		loans = append(loans, loan)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return loans, nil
}

func (d *DebtRepoMysql) FindActiveDebts(debtorID int) ([]model.Debt, error) {
	// az == debtor & status == ongoing
	statement := `SELECT creditor, amount, description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.debtor = ? AND s.status=?`
	rows, err := d.db.Query(statement, debtorID, ongoingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []model.Debt{}
	for rows.Next() {
		var debt model.Debt
		err := rows.Scan(&debt.CreditorID, &debt.Amount, &debt.Description)
		if err != nil {
			return nil, err
		}
		debts = append(debts, debt)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return debts, nil
}

func (d *DebtRepoMysql) FindPendingLoans(creditorID int) ([]model.Loan, error) {
	// az == creditor & status == ongoing
	statement := `SELECT debtor, amount, description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.creditor = ? AND s.status = ?`
	rows, err := d.db.Query(statement, creditorID, pendingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := []model.Loan{}
	for rows.Next() {
		var loan model.Loan
		err := rows.Scan(&loan.DebtorID, &loan.Amount, &loan.Description)
		if err != nil {
			return nil, err
		}
		loans = append(loans, loan)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return loans, nil
}

func (d *DebtRepoMysql) FindPendingDebts(debtorID int) ([]model.Debt, error) {
	statement := `SELECT creditor, amount, description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.debtor = ? AND s.status=?`
	rows, err := d.db.Query(statement, debtorID, pendingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []model.Debt{}
	for rows.Next() {
		var debt model.Debt
		err := rows.Scan(&debt.CreditorID, &debt.Amount, &debt.Description)
		if err != nil {
			return nil, err
		}
		debts = append(debts, debt)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return debts, nil
}

// todo
// say that someone has returned the money
