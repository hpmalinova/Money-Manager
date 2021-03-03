package contract

import "github.com/hpmalinova/Money-Manager/model"

type UserRepo interface {
	Find(start, count int) ([]model.User, error)
	FindByID(id int) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Create(user *model.User) (*model.User, error)
}
