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

func (p *PaymentRepoMysql) CheckBalance(userID int) (int, error) {
	var balance int
	statement := "SELECT balance FROM users WHERE user_id= ?"
	err := p.db.QueryRow(statement, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (p *PaymentRepoMysql) Pay(pay *model.History) error {
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

	// Pay
	statement := "INSERT INTO money_history(uid, amount, category_id, description) VALUES(?, ?, ?, ?)"
	_, err = tx.ExecContext(ctx, statement, pay.UserID, pay.Amount, pay.CategoryID, pay.Description)
	if err != nil {
		return err
	}

	// Decrease wallet
	// TODO check if works for negative balance
	statement = "UPDATE wallet SET balance = balance - ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, pay.Amount, pay.UserID)
	if err != nil {
		msg := fmt.Sprintf("not enough money: %s", err.Error())
		return errors.New(msg)
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		msg := fmt.Sprintf("error in paying: %s\n", err)
		return errors.New(msg)
	}
	return nil
}

func (p *PaymentRepoMysql) Earn(pay *model.History) error {
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
	_, err = tx.ExecContext(ctx, statement, pay.UserID, pay.Amount, pay.CategoryID, pay.Description)
	if err != nil {
		return err
	}

	statement = "UPDATE wallet SET balance = balance + ? WHERE user_id = ?"
	_, err = tx.ExecContext(ctx, statement, pay.Amount, pay.UserID)
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
