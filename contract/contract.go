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

type GroupRepo interface {
	Create(name string, participants []int) error
	Find(start, count, ownerID int) ([]model.Group, error)
}

type CategoryRepo interface {
	FindByName(categoryName string) (*model.Category, error)
	FindExpenses() ([]model.Category, error)
	FindIncomes() ([]model.Category, error)
	FindAll() ([]model.Category, error)
}

type PaymentRepo interface {
	CheckBalance(userID int) (int, error)

	Pay(h *model.History) error
	Earn(h *model.History) error
	GiveLoan(t *model.Transfer) error
	Split(t *model.Transfer) error

	FindActiveDebts(debtorID int) ([]model.Debt, error)
	FindActiveLoans(creditorID int) ([]model.Loan, error)
	RequestRepay(debtID, amount int) error
}
