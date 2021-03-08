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
	Find() ([]model.Category, error)
}

type HistoryRepo interface {
	Pay(history *model.History) error
	GiveLoan(loan *model.LoanHistory) error
	ReceiveDebt(loan *model.LoanHistory) error
}

type DebtRepo interface {
	Add(debt *model.DebtAndLoan) error
	RequestPaymentConfirmation(statusID, amount int) error
	AcceptPayment(statusID int) (int, error)
	DeclinePayment(statusID int) error
	FindActiveLoans(creditorID int) ([]model.Loan, error)
	FindActiveDebts(debtorID int) ([]model.Debt, error)
	FindPendingLoans(creditorID int) ([]model.Loan, error)
	FindPendingDebts(debtorID int) ([]model.Debt, error)
}
