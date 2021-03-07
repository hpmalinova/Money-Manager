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

type GroupRepoMysql struct {
	db *sql.DB
}

func NewGroupRepoMysql(user, password, dbname string) *GroupRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &GroupRepoMysql{}
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

func (g *GroupRepoMysql) CreateGroup(name string, participants []int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := g.db.Conn(ctx)
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

	statement, err := tx.Prepare(`INSERT INTO groups(name, participant_id) VALUES( ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()

	for uid := range participants {
		result, err := statement.Exec(name, uid)
		if err != nil {
			prefix := "Bad Request: "
			msg := fmt.Sprintf("%sUser: %d already participates in %s", prefix, uid, name)
			return errors.New(msg)
		}
		numRows, err := result.RowsAffected()
		if err != nil || numRows != 1 {
			msg := fmt.Sprintf("error inserting participantID: %v, %s\n", uid, err)
			return errors.New(msg)
		}
	}

	// COMMIT TRANSACTION
	if err := tx.Commit(); err != nil {
		msg := fmt.Sprintf("error in creating group: %s\n", err)
		return errors.New(msg)
	}
	return nil
}

func (g *GroupRepoMysql) Find(start, count, ownerID int) ([]model.Group, error) {
	statement := `SELECT id, name, participant_id FROM groups 
					WHERE participant_id = ?
					LIMIT ? OFFSET ?` // TODO fix stmt // change to groupID-groupNAME + groupID-userID

	rows, err := g.db.Query(statement, ownerID, count, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []model.Group{}
	for rows.Next() {
		var group model.Group
		err := rows.Scan(&group.ID, &group.Name, &group.ParticipantIDs)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}
