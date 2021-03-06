package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/hpmalinova/Money-Manager/model"
	"log"
)

const (
	pending  = "pending"
	accepted = "accepted"
	declined = "declined"
)

type FriendshipRepoMysql struct {
	db *sql.DB
}

func NewFriendRepoMysql(user, password, dbname string) *FriendshipRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &FriendshipRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	return repo
}

func (f *FriendshipRepoMysql) Add(friends *model.Friendship) error {
	if f.checkFriendship(friends.UserOne, friends.UserTwo) {
		return errors.New("you are already friends")
	}
	statement := "INSERT INTO friendship(user_one_id, user_two_id, status, action_user_id) VALUES(?, ?, ?, ?)"
	_, err := f.db.Exec(statement, friends.UserOne, friends.UserTwo, pending, friends.ActionUser)
	if err != nil {
		return err
	}

	return nil
}

func (f *FriendshipRepoMysql) checkFriendship(userOne, userTwo int) bool {
	statement := "SELECT * FROM friendship WHERE user_one_id = ? AND user_two_id = ? AND status = ?"
	err := f.db.QueryRow(statement, userOne, userTwo, accepted).Scan()
	if err != nil {
		return false
	}
	return true
}

func (f FriendshipRepoMysql) Find(start, count, userID int) ([]int, error) {
	statement := `SELECT user_one_id, user_two_id FROM friendship 
					WHERE (user_one_id = ? OR user_two_id = ?) AND status = ?
					LIMIT ? OFFSET ?`
	rows, err := f.db.Query(statement, userID, userID, accepted, count, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []int{}
	for rows.Next() {
		var userOne, userTwo int
		err := rows.Scan(&userOne, &userTwo)
		if err != nil {
			return nil, err
		}

		var friendID int
		if userOne != userID {
			friendID = userOne
		} else {
			friendID = userTwo
		}

		friends = append(friends, friendID)
	}
	rows.Close()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return friends, nil
}

func (f FriendshipRepoMysql) FindPending(start, count, userID int) ([]int, error) {
	statement := `SELECT user_one_id, user_two_id FROM friendship 
					WHERE (user_one_id = ? OR user_two_id = ?) AND status = ? AND action_user_id != ?
					LIMIT ? OFFSET ?`
	rows, err := f.db.Query(statement, userID, userID, pending, userID, count, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []int{}
	for rows.Next() {
		var userOne, userTwo int
		err := rows.Scan(&userOne, &userTwo)
		if err != nil {
			return nil, err
		}

		var friendID int
		if userOne != userID {
			friendID = userOne
		} else {
			friendID = userTwo
		}

		friends = append(friends, friendID)
	}
	rows.Close()

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return friends, nil
}

func (f FriendshipRepoMysql) AcceptInvite(userOne, userTwo, actionUser int) error {
	statement := "UPDATE friendship SET status = ?, action_user_id = ?  WHERE `user_one_id` = ? AND `user_two_id` = ?"
	return f.db.QueryRow(statement, accepted, actionUser, userOne, userTwo).Scan()
}

func (f FriendshipRepoMysql) DeclineInvite(userOne, userTwo, actionUser int) error {
	statement := "UPDATE friendship SET status = ?, action_user_id = ?  WHERE `user_one_id` = ? AND `user_two_id` = ?"
	return f.db.QueryRow(statement, declined, actionUser, userOne, userTwo).Scan()
}