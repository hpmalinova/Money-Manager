package repository

import (
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hpmalinova/Money-Manager/model"
	"reflect"
	"testing"
)


func TestFriendshipRepoMysql_Add(t *testing.T) {
	db, mock := NewMock()
	statement := "INSERT INTO friendship"
	mock.ExpectExec(statement).WithArgs(1,2,"pending", 1).WillReturnResult(sqlmock.NewResult(1, 1))

	db2, mock2 := NewMock()
	mock2.ExpectExec(statement).WithArgs(1,2,"pending", 1).WillReturnError(errors.New("error"))

	type fields struct {
		db *sql.DB
	}
	type args struct {
		friends *model.Friendship
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db},
			args: struct{ friends *model.Friendship }{friends: &model.Friendship{
				UserOne:    1,
				UserTwo:    2,
				Status:     "pending",
				ActionUser: 1,
			}},
			wantErr: false,
		},
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db2},
			args: struct{ friends *model.Friendship }{friends: &model.Friendship{
				UserOne:    1,
				UserTwo:    2,
				Status:     "pending",
				ActionUser: 1,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FriendshipRepoMysql{
				db: tt.fields.db,
			}
			if err := f.Add(tt.args.friends); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFriendshipRepoMysql_AcceptInvite(t *testing.T) {
	db, mock := NewMock()
	statement := "UPDATE friendship"
	mock.ExpectExec(statement).WithArgs("accepted", 1,1,2).WillReturnResult(sqlmock.NewResult(0, 1))

	db2, mock2 := NewMock()
	mock2.ExpectExec(statement).WithArgs("accepted", 1,1,2).WillReturnError(errors.New("error"))

	type fields struct {
		db *sql.DB
	}
	type args struct {
		userOne    int
		userTwo    int
		actionUser int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db},
			args: struct {
				userOne    int
				userTwo    int
				actionUser int
			}{userOne: 1, userTwo: 2, actionUser: 1},
			wantErr: false,
		},
		{
			name: "fail",
			fields: struct{ db *sql.DB }{db: db2},
			args: struct {
				userOne    int
				userTwo    int
				actionUser int
			}{userOne: 1, userTwo: 2, actionUser: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FriendshipRepoMysql{
				db: tt.fields.db,
			}
			if err := f.AcceptInvite(tt.args.userOne, tt.args.userTwo, tt.args.actionUser); (err != nil) != tt.wantErr {
				t.Errorf("AcceptInvite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFriendshipRepoMysql_DeclineInvite(t *testing.T) {
	db, mock := NewMock()
	statement := "DELETE FROM friendship"
	mock.ExpectExec(statement).WithArgs(1,2).WillReturnResult(sqlmock.NewResult(0, 1))

	db2, mock2 := NewMock()
	mock2.ExpectExec(statement).WithArgs(1,2).WillReturnError(errors.New("error"))

	type fields struct {
		db *sql.DB
	}
	type args struct {
		userOne int
		userTwo int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db},
			args: struct {
				userOne int
				userTwo int
			}{userOne: 1, userTwo: 2},
			wantErr: false,
		},
		{
			name: "fail",
			fields: struct{ db *sql.DB }{db: db2},
			args: struct {
				userOne int
				userTwo int
			}{userOne: 1, userTwo: 2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FriendshipRepoMysql{
				db: tt.fields.db,
			}
			if err := f.DeclineInvite(tt.args.userOne, tt.args.userTwo); (err != nil) != tt.wantErr {
				t.Errorf("DeclineInvite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFriendshipRepoMysql_Find(t *testing.T) {
	db, mock := NewMock()
	statement := "SELECT user_one_id, user_two_id FROM friendship"
	rows := sqlmock.NewRows([]string{"user_one_id", "user_two_id"}).
		AddRow(1,2)
	mock.ExpectQuery(statement).WithArgs(1,1,accepted, 10, 0).WillReturnRows(rows)

	db2, mock2 := NewMock()
	mock2.ExpectQuery(statement).WithArgs(1,1,accepted, 10, 0).WillReturnError(errors.New("error"))

	type fields struct {
		db *sql.DB
	}
	type args struct {
		start  int
		count  int
		userID int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db},
			args: struct {
				start  int
				count  int
				userID int
			}{start: 0, count: 10, userID: 1},
			want: []int{2},
			wantErr: false,
		},
		{
			name: "fail",
			fields: struct{ db *sql.DB }{db: db2},
			args: struct {
				start  int
				count  int
				userID int
			}{start: 0, count: 10, userID: 1},
			want: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FriendshipRepoMysql{
				db: tt.fields.db,
			}
			got, err := f.Find(tt.args.start, tt.args.count, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Find() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFriendshipRepoMysql_FindPending(t *testing.T) {
	db, mock := NewMock()
	statement := "SELECT user_one_id, user_two_id FROM friendship"
	rows := sqlmock.NewRows([]string{"user_one_id", "user_two_id"}).
		AddRow(1,2)
	mock.ExpectQuery(statement).WithArgs(1,1,pending, 1, 10, 0).WillReturnRows(rows)

	db2, mock2 := NewMock()
	mock2.ExpectQuery(statement).WithArgs(1,1,pending, 1, 10, 0).WillReturnError(errors.New("error"))

	type fields struct {
		db *sql.DB
	}
	type args struct {
		start  int
		count  int
		userID int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "success",
			fields: struct{ db *sql.DB }{db: db},
			args: struct {
				start  int
				count  int
				userID int
			}{start: 0, count: 10, userID: 1},
			want: []int{2},
			wantErr: false,
		},
		{
			name: "fail",
			fields: struct{ db *sql.DB }{db: db2},
			args: struct {
				start  int
				count  int
				userID int
			}{start: 0, count: 10, userID: 1},
			want: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FriendshipRepoMysql{
				db: tt.fields.db,
			}
			got, err := f.FindPending(tt.args.start, tt.args.count, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindPending() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindPending() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//func TestFriendshipRepoMysql_checkFriendship(t *testing.T) {
//	db3, mock3 := NewMock()
//	statement2 := "SELECT user_one_id, user_two_id, status, action_user_id FROM friendship"
//	rows := sqlmock.NewRows([]string{"user_one_id", "user_two_id", "status", "action_user_id"}).
//		AddRow(1,2,"accepted",1)
//	mock3.ExpectQuery(statement2).WithArgs(1,2,"accepted").WillReturnRows(rows)
//
//	type fields struct {
//		db *sql.DB
//	}
//	type args struct {
//		userOne int
//		userTwo int
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		{
//			name: "already friends",
//			fields: struct{ db *sql.DB }{db: db3},
//			args: struct {
//				userOne int
//				userTwo int
//			}{userOne: 1, userTwo: 2},
//			want: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			f := &FriendshipRepoMysql{
//				db: tt.fields.db,
//			}
//			if got := f.checkFriendship(tt.args.userOne, tt.args.userTwo); got != tt.want {
//				t.Errorf("checkFriendship() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}