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

type PaymentRepoMysql struct {
	db *sql.DB
}

func NewPaymentRepoMysql(user, password, dbname string) *PaymentRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &PaymentRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	repo.db.SetConnMaxLifetime(time.Minute * 5)
	repo.db.SetMaxOpenConns(10)
	repo.db.SetMaxIdleConns(10)
	repo.db.SetConnMaxIdleTime(time.Minute * 3)

	return repo
}

const (
	ongoingStatus = "ongoing"
	pendingStatus = "pending"
)

func (p *PaymentRepoMysql) CheckBalance(userID int) (int, error) {
	var balance int
	statement := "SELECT balance FROM wallet WHERE user_id= ?"
	err := p.db.QueryRow(statement, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (p *PaymentRepoMysql) CreateWallet(userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	statement := "INSERT INTO wallet(user_id, balance) VALUES(?, ?)"
	_, err := p.db.ExecContext(ctx, statement, userID, 0)
	if err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) Pay(h *model.History) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	// Decrease wallet
	// TODO check if works for negative balance
	statement := "UPDATE wallet SET balance = balance - ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, h.Amount, h.UserID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// Pay
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, h.UserID, h.Amount, h.CategoryID, h.Description)
	if err != nil {
		return err
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) Earn(h *model.History) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	statement := "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, h.UserID, h.Amount, h.CategoryID, h.Description)
	if err != nil {
		return err
	}

	statement = "UPDATE wallet SET balance = balance + ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, h.Amount, h.UserID)
	if err != nil {
		return err
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) GiveLoan(t *model.TransferLoan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	// Remove money from wallet (Creditor)
	statement := "UPDATE wallet SET balance = balance - ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, t.Amount, t.CreditorID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// Add money to wallet (Debtor)
	statement = "UPDATE wallet SET balance = balance + ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, t.Amount, t.DebtorID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// Add to expenses (Creditor)
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, t.CreditorID, t.Amount, t.LoanCategoryID, t.Description)
	if err != nil {
		return err
	}

	// Add to incomes (Debtor)
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, t.DebtorID, t.Amount, t.DebtCategoryID, t.Description)
	if err != nil {
		return err
	}

	// Add Debt
	statement = "INSERT INTO debt_status(status, amount) VALUES(?, ?)"
	result, err := tx.ExecContext(ctx, statement, ongoingStatus, t.Amount)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	statusID := int(id)

	statement = "INSERT INTO debts(creditor, debtor, amount, category, description, status_id) VALUES(?, ?, ?, ?, ?, ?)"
	result, err = tx.ExecContext(ctx, statement, t.CreditorID, t.DebtorID, t.Amount, t.RepayCategoryName, t.Description, statusID)
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
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) Split(t *model.TransferSplit) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	// Remove money from wallet (Creditor)
	statement := "UPDATE wallet SET balance = balance - ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, t.Amount, t.CreditorID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	halfAmount := t.Amount / 2

	// Add to expenses (Creditor: Pay)
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, t.CreditorID, halfAmount, t.Expense.ID, t.Description)
	if err != nil {
		return err
	}

	// Add to expenses (Creditor: Loan)
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, t.CreditorID, halfAmount, t.LoanCategoryID, t.Description)
	if err != nil {
		return err
	}

	// Add Debt
	statement = "INSERT INTO debt_status(status, amount) VALUES(?, ?)"
	result, err := tx.ExecContext(ctx, statement, ongoingStatus, halfAmount)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	statusID := int(id)

	statement = "INSERT INTO debts(creditor, debtor, amount, category, description, status_id) VALUES(?, ?, ?, ?, ?, ?)"
	result, err = tx.ExecContext(ctx, statement, t.CreditorID, t.DebtorID, halfAmount, t.Expense.Name, t.Description, statusID)
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
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) FindActiveDebts(debtorID int) ([]model.DebtExt, error) {
	statement := `SELECT d.status_id, d.creditor, d.amount, d.description, d.category 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.debtor = ? AND s.status=?`
	rows, err := p.db.Query(statement, debtorID, ongoingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	debts := []model.DebtExt{}
	for rows.Next() {
		var debt model.DebtExt
		err := rows.Scan(&debt.StatusID, &debt.CreditorID, &debt.Amount, &debt.Description, &debt.CategoryName)
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

func (p *PaymentRepoMysql) FindActiveLoans(creditorID int) ([]model.Loan, error) {
	statement := `SELECT d.debtor, d.amount, d.description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.creditor = ? AND s.status=?`
	rows, err := p.db.Query(statement, creditorID, ongoingStatus)
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

func (p *PaymentRepoMysql) RequestRepay(debtID, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	// Get Amount of Debt
	statement := "SELECT amount FROM debt_status WHERE id = ?"
	var debtAmount int
	err = tx.QueryRowContext(ctx, statement, debtID).Scan(&debtAmount)
	if err != nil {
		return err
	}

	// You can`t repay more than you've received
	if amount > debtAmount {
		amount = debtAmount
	}

	statement = "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
	if _, err = tx.ExecContext(ctx, statement, pendingStatus, amount, debtID); err != nil {
		return err
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) FindPendingDebts(debtorID int) ([]model.Debt, error) {
	statement := `SELECT d.creditor, s.amount, d.description 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.debtor = ? AND s.status=?`
	rows, err := p.db.Query(statement, debtorID, pendingStatus)
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

func (p *PaymentRepoMysql) FindPendingRequests(creditorID int) ([]model.LoanExt, error) {
	statement := `SELECT d.debtor, s.amount, d.description, d.status_id 
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE d.creditor = ? AND s.status = ?`
	rows, err := p.db.Query(statement, creditorID, pendingStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := []model.LoanExt{}
	for rows.Next() {
		var loan model.LoanExt
		err := rows.Scan(&loan.DebtorID, &loan.Amount, &loan.Description, &loan.StatusID)
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

func (p *PaymentRepoMysql) AcceptPayment(a *model.Accept) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	ap := model.AcceptPayment{}
	statement := `SELECT d.creditor, d.debtor, d.amount, d.description, s.amount
					FROM debts AS d
					INNER JOIN debt_status AS s
						ON d.status_id = s.id
					WHERE s.id=?`
	err = tx.QueryRowContext(ctx, statement, a.StatusID).Scan(&ap.CreditorID, &ap.DebtorID, &ap.DebtAmount,
		&ap.Description, &ap.PendingAmount)
	if err != nil {
		return err
	}

	// Remove money from Debtor`s wallet
	statement = "UPDATE wallet SET balance = balance - ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, ap.PendingAmount, ap.DebtorID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// Receive money
	statement = "UPDATE wallet SET balance = balance + ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, ap.PendingAmount, ap.CreditorID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// Update debt
	if ap.PendingAmount < ap.DebtAmount {
		// Decrease the debt
		statement = "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
		_, err = tx.ExecContext(ctx, statement, ongoingStatus, ap.DebtAmount-ap.PendingAmount, a.StatusID)
		if err != nil {
			return err
		}

		statement = "UPDATE debts SET amount = ? WHERE status_id = ?"
		_, err = tx.ExecContext(ctx, statement, ap.DebtAmount-ap.PendingAmount, a.StatusID)
		if err != nil {
			return err
		}
	} else if ap.PendingAmount == ap.DebtAmount {
		// Delete the debt
		statement := "DELETE FROM debt_status WHERE id=?"
		if _, err := tx.ExecContext(ctx, statement, a.StatusID); err != nil {
			return err
		}

		statement = "DELETE FROM debts WHERE status_id=?"
		if _, err := tx.ExecContext(ctx, statement, a.StatusID); err != nil {
			return err
		}
	}

	// Update History

	// Creditor
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, ap.CreditorID, ap.PendingAmount, a.RepayC.ID, ap.Description)
	if err != nil {
		return err
	}

	// Debtor
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, ap.DebtorID, ap.PendingAmount, a.ExpenseC.ID, ap.Description)
	if err != nil {
		return err
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) DeclinePayment(statusID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := p.db.Conn(ctx)
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

	statement := "SELECT amount FROM debts WHERE status_id = ?"
	var debtAmount int
	if err := tx.QueryRowContext(ctx, statement, statusID).Scan(&debtAmount); err != nil {
		return err
	}

	statement = "UPDATE debt_status SET status = ?, amount = ? WHERE id = ?"
	if _, err = tx.ExecContext(ctx, statement, ongoingStatus, debtAmount, statusID); err != nil {
		return err
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PaymentRepoMysql) FindCategoryName(statusID int) (categoryName string, err error) {
	statement := `SELECT category FROM debts WHERE status_id=?`
	err = p.db.QueryRow(statement, statusID).Scan(&categoryName)
	return categoryName, err
}

func (p *PaymentRepoMysql) FindHistory(userID int) (*model.HistoryShowAll, error) {
	aps := []model.HistoryShow{}
	statement := `SELECT m.amount, m.description, c.c_type, c.name
					FROM money_history AS m
					INNER JOIN categories AS c
						ON m.category_id = c.id
					WHERE m.uid=?`
	results, err := p.db.Query(statement, userID)
	if err != nil {
		return nil, err
	}

	for results.Next() {
		ap := model.HistoryShow{}
		err = results.Scan(&ap.Amount, &ap.Description, &ap.CategoryType, &ap.CategoryName)
		if err != nil {
			return nil, err
		}
		aps = append(aps, ap)
	}
	return &model.HistoryShowAll{HistoryShowAll: aps}, nil
}

// t: true == "expense" or false == "income"
func (p *PaymentRepoMysql) FindStatistics(userID int, t bool) (*model.Statistics, error) {
	statement := `SELECT COALESCE(SUM(amount),0)
					FROM money_history as m
					JOIN categories as c 
						ON m.category_id=c.id
					WHERE uid=? AND c.c_type=?;`
	var cType string
	if t {
		cType = "expense"
	} else {
		cType = "income"
	}
	var sum int
	err := p.db.QueryRow(statement, userID, cType).Scan(&sum)
	if err != nil {
		return nil, err
	}

	if sum <= 0 {
		rs := []model.Ratio{}
		r := model.Ratio{
			Percent:      "0",
			CategoryName: "No "+cType+"s",
		}
		rs = append(rs, r)
		return &model.Statistics{Ratios: rs}, nil
	}

	statement = `SELECT c.name, SUM(amount) 
					FROM money_history as m
					JOIN categories as c 
						ON m.category_id=c.id
					WHERE uid=? AND c.c_type=?
					group by c.name;`
	results, err := p.db.Query(statement, userID, cType)
	if err != nil {
		return nil, err
	}

	rs := []model.Ratio{}
	for results.Next() {
		r := model.Ratio{}
		var s int
		err = results.Scan(&r.CategoryName, &s)
		if err != nil {
			return nil, err
		}
		percent := float64(s) / float64(sum)
		r.Percent = fmt.Sprintf("%.2f", percent)
		rs = append(rs, r)
	}
	return &model.Statistics{Ratios: rs}, nil
}