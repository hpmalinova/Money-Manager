package contract

import "github.com/hpmalinova/Money-Manager/model"

type UserRepo interface {
	Find(start, count int) ([]model.User, error)
	FindByID(id int) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Create(user *model.User) (*model.User, error)
}

type FriendshipRepo interface {
	Add(friendship *model.Friendship) error
	Find(start, count, userID int) ([]int, error)
	FindPending(start, count, userID int) ([]int, error)
	AcceptInvite(userOne, userTwo, actionUser int) error
	DeclineInvite(userOne, userTwo, actionUser int) error
}
