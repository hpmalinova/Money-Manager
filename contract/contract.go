package contract

import "github.com/hpmalinova/Money-Manager/model"

type UserRepo interface {
	Find(start, count int) ([]model.User, error)
	FindByID(id int) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindNamesByIDs(ids []int) ([]string, error)
	Create(user *model.User) (*model.User, error)
}

type FriendshipRepo interface {
	Add(friendship *model.Friendship) error
	Find(start, count, userID int) ([]int, error)
	FindPending(start, count, userID int) ([]int, error)
	AcceptInvite(userOne, userTwo, actionUser int) error
	DeclineInvite(userOne, userTwo int) error
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
	CreateWallet(userID int) error

	Pay(h *model.History) error
	Earn(h *model.History) error
	GiveLoan(t *model.TransferLoan) error
	Split(t *model.TransferSplit) error

	FindActiveDebts(debtorID int) ([]model.DebtExt, error)
	FindActiveLoans(creditorID int) ([]model.Loan, error)
	RequestRepay(debtID, amount int) error

	FindPendingDebts(debtorID int) ([]model.Debt, error)
	FindPendingRequests(creditorID int) ([]model.LoanExt, error)

	AcceptPayment(a *model.Accept) error
	DeclinePayment(statusID int) error

	FindHistory(userID int) (*model.HistoryShowAll, error)
	FindStatistics(userID int, t bool) (*model.Statistics, error)

	FindCategoryName(statusID int) (categoryName string, err error)
}
