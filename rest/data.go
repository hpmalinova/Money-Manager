package rest

import (
	"github.com/hpmalinova/Money-Manager/model"
	"golang.org/x/crypto/bcrypt"
)

var userIDs = []int{1, 2, 3, 4}

func (a *App) AddData() {
	pass1, _ := bcrypt.GenerateFromPassword([]byte("love"), bcrypt.DefaultCost)
	pass2, _ := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.DefaultCost)

	_, _ = a.Users.Create(&model.User{ID: 1, Username: "Hrisi", Password: string(pass1)})
	_, _ = a.Users.Create(&model.User{ID: 2, Username: "Peter", Password: string(pass2)})
	_, _ = a.Users.Create(&model.User{ID: 3, Username: "George", Password: string(pass2)})
	_, _ = a.Users.Create(&model.User{ID: 4, Username: "Lily", Password: string(pass2)})

	a.addMoneyToWallet(100)
	a.addFriendships()
	a.addPayments()
	a.addLoans()
	a.addSplit()
}

func (a *App) addMoneyToWallet(amount int) {
	for _, id := range userIDs {
		_ = a.Payment.CreateWallet(id)
		_ = a.Payment.Earn(&model.History{
			UserID:      id,
			Amount:      amount,
			CategoryID:  8,
			Description: "",
		})
	}
}

// Peter --> Hrisi (pending)
// Hrisi --> George (pending)
// Peter+George
// Hrisi+Lily
func (a *App) addFriendships() {
	_ = a.Friendship.Add(&model.Friendship{
		UserOne:    1,
		UserTwo:    2,
		ActionUser: 2,
	})
	_ = a.Friendship.Add(&model.Friendship{
		UserOne:    1,
		UserTwo:    3,
		ActionUser: 1,
	})
	_ = a.Friendship.Add(&model.Friendship{
		UserOne:    2,
		UserTwo:    3,
		ActionUser: 3,
	})
	_ = a.Friendship.AcceptInvite(2, 3, 2)
	_ = a.Friendship.Add(&model.Friendship{
		UserOne:    1,
		UserTwo:    4,
		ActionUser: 4,
	})
	_ = a.Friendship.AcceptInvite(1, 4, 1)
}

// Hrisi: 70
// Peter: 90
// George: 80
// Lily: 10
func (a *App) addPayments() {
	amount := []int{5, 10, 20, 90}
	category := []int{3, 4, 5, 3}
	description := []string{"Bread", "", "Car Wash", "Bar"}

	for i, id := range userIDs {
		_ = a.Payment.Pay(&model.History{
			UserID:      id,
			Amount:      amount[i],
			CategoryID:  category[i],
			Description: description[i],
		})
	}

	_ = a.Payment.Pay(&model.History{
		UserID:      1,
		Amount:      15,
		CategoryID:  3,
		Description: "",
	})
	_ = a.Payment.Pay(&model.History{
		UserID:      1,
		Amount:      10,
		CategoryID:  4,
		Description: "",
	})
}

// Hrisi: 40
// Peter: 90
// George: 80
// Lily: 40
// Hrisi --> Lily (30lv "Bills")
func (a *App) addLoans() {
	_ = a.Payment.GiveLoan(&model.TransferLoan{
		DebtCategoryID:    6,
		RepayCategoryName: "repay",
		Transfer: model.Transfer{
			CreditorID:     1,
			LoanCategoryID: 1,
			Loan: model.Loan{
				DebtorID:    4,
				Amount:      30,
				Description: "Bills",
			},
		},
	})
}

// Hrisi: 40
// Peter: 90
// George: 20
// Lily: 40
// George --> Peter 60 FOOD "Restaurant"
func (a *App) addSplit() {
	_ = a.Payment.Split(&model.TransferSplit{
		Expense: model.Category{
			ID:   3,
			Name: "food",
		},
		Transfer: model.Transfer{
			CreditorID:     3,
			LoanCategoryID: 1,
			Loan: model.Loan{
				DebtorID:    2,
				Amount:      60,
				Description: "Restaurant",
			},
		},
	})
}
