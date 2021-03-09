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
	statement := "SELECT balance FROM users WHERE user_id= ?"
	err := p.db.QueryRow(statement, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
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
		msg := fmt.Sprintf("error in paying: %s\n", err)
		return errors.New(msg)
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
		msg := fmt.Sprintf("error in earning: %s\n", err)
		return errors.New(msg)
	}
	return nil
}

func (p *PaymentRepoMysql) GiveLoan(t *model.Transfer) error {
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
	_, err = tx.ExecContext(ctx, statement, t.CreditorID, t.Amount, t.LoanID, t.Description)
	if err != nil {
		return err
	}

	// Add to incomes (Debtor)
	statement = "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, t.DebtorID, t.Amount, t.DebtID, t.Description)
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
	result, err = tx.ExecContext(ctx, statement, t.CreditorID, t.DebtorID, t.Amount, t.DebtID, t.Description, statusID)
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
		msg := fmt.Sprintf("error in giving loan: %s\n", err)
		return errors.New(msg)
	}
	return nil
}
