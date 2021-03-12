package rest

import (
	"fmt"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gorilla/mux"
	"github.com/hpmalinova/Money-Manager/contract"
	"github.com/hpmalinova/Money-Manager/model"
	"github.com/hpmalinova/Money-Manager/repository"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

type App struct {
	Router *mux.Router

	Users      contract.UserRepo
	Friendship contract.FriendshipRepo
	Groups     contract.GroupRepo
	Categories contract.CategoryRepo
	Payment    contract.PaymentRepo

	Validator  *validator.Validate
	Translator ut.Translator
	Template   *template.Template
}

func (a *App) Init(user, password, dbname string) {
	// db=sqlopen
	// newrepo(&db) --> repo.db = db
	a.Users = repository.NewUserRepoMysql(user, password, dbname) // TODO one db connection?
	a.Friendship = repository.NewFriendRepoMysql(user, password, dbname)
	a.Groups = repository.NewGroupRepoMysql(user, password, dbname)
	a.Categories = repository.NewCategoryRepoMysql(user, password, dbname)
	a.Payment = repository.NewPaymentRepoMysql(user, password, dbname)

	a.Validator = validator.New()
	eng := en.New()
	var uni *ut.UniversalTranslator
	uni = ut.New(eng, eng)

	var found bool
	a.Translator, found = uni.GetTranslator("en")
	if !found {
		log.Fatal("translator not found")
	}
	if err := en_translations.RegisterDefaultTranslations(a.Validator, a.Translator); err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.Template = template.Must(template.ParseGlob("templates/*"))
	a.initializeRoutes()
}

func (a *App) Run(port string) {
	log.Fatal(http.ListenAndServe(":"+port, a.Router))
}

const (
	welcome  = "welcome"
	register = "register"
	login    = "login"

	index  = "index"
	logout = "logout"

	users   = "users"
	friends = "friends"

	earn     = "earn"
	pay      = "pay"
	giveLoan = "loan"
	split    = "split"

	debts = "debts"
	repay = "repay"

	loans = "loans"
)

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/", a.welcome).Methods(http.MethodGet)
	a.Router.HandleFunc("/"+register, a.registerHandler).Methods(http.MethodGet, http.MethodPost)
	a.Router.HandleFunc("/"+login, a.loginHandler).Methods(http.MethodGet, http.MethodPost)

	// Auth route
	s := a.Router.PathPrefix("/" + index).Subrouter()
	s.Use(JwtVerify) // Middleware
	s.HandleFunc("", a.index).Methods(http.MethodGet)
	s.HandleFunc("/"+logout, a.logout).Methods(http.MethodPost)
	s.HandleFunc("/"+users, a.getUsers).Methods(http.MethodGet)
	s.HandleFunc("/"+friends, a.getFriends).Methods(http.MethodGet, http.MethodPost)
	s.HandleFunc("/"+friends+"/accept/{username}", a.acceptInvite).Methods(http.MethodPost)
	s.HandleFunc("/"+friends+"/decline/{username}", a.declineInvite).Methods(http.MethodPost)
	s.HandleFunc("/"+friends+"/add", a.addFriend).Methods(http.MethodPost)

	s.HandleFunc("/"+earn, a.earn).Methods(http.MethodGet, http.MethodPost)
	s.HandleFunc("/"+pay, a.pay).Methods(http.MethodGet, http.MethodPost)
	s.HandleFunc("/"+giveLoan, a.giveLoan).Methods(http.MethodPost)
	s.HandleFunc("/"+split, a.split).Methods(http.MethodPost)

	s.HandleFunc("/"+debts, a.getDebts).Methods(http.MethodGet)
	s.HandleFunc("/"+debts+"/"+repay+"/{id:[0-9]+}", a.requestRepay).Methods(http.MethodPost)

	//s.HandleFunc("/"+loans, a.getLoans).Methods(http.MethodGet)
}

func (a *App) welcome(w http.ResponseWriter, r *http.Request) {
	_ = a.Template.ExecuteTemplate(w, welcome, nil)
}

func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+register {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		_ = a.Template.ExecuteTemplate(w, register, nil)
	case "POST":
		if err := r.ParseForm(); err != nil {
			_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		user := &model.User{Username: username, Password: password}

		// Validate User struct
		err := a.Validator.Struct(user)
		if err != nil {
			errs := err.(validator.ValidationErrors)
			respondWithValidationError(errs.Translate(a.Translator), w)
			return
		}

		// Hash the password with bcrypt
		pass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println(err)
			respondWithError(w, http.StatusInternalServerError, "Password Encryption  failed")
			return
		}
		user.Password = string(pass)

		if user, err = a.Users.Create(user); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Create wallet
		err = a.Payment.CreateWallet(user.ID)
		log.Println(err)

		http.Redirect(w, r, "/", http.StatusFound)
	default:
		_, _ = fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+login {
		fmt.Println(r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		a.Template.ExecuteTemplate(w, login, nil)
	case "POST":
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		user := &model.UserLogin{Username: username, Password: password}

		err := a.Validator.Struct(user)
		if err != nil {
			errs := err.(validator.ValidationErrors)
			respondWithValidationError(errs.Translate(a.Translator), w)
			return
		}

		resp, err := a.checkCredentials(w, user.Username, user.Password)
		if err == nil {
			tokenString := resp["token"]
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    tokenString,
				Expires:  time.Now().Add(30 * time.Minute),
				HttpOnly: true,
				//MaxAge: 60*60,
			})
		}

		http.Redirect(w, r, "/"+index, http.StatusFound)
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value("user").(*model.UserToken)
	userID, _ := strconv.Atoi(user.UserID)
	// Show balance
	balance, _ := a.Payment.CheckBalance(userID)

	_ = a.Template.ExecuteTemplate(w, index, model.UserWallet{
		Username: user.Username,
		Balance:  balance,
	})
}

//todo
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println("logout")
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		//MaxAge: 0,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	// TODO
	//count, err, start, done := a.getStartCount(w, r)
	//if done {
	//	return
	//}
	start, count := 0, 10

	users, err := a.Users.Find(start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// remove user passwords
	for i := range users {
		users[i].Password = ""
	}

	a.Template.ExecuteTemplate(w, "showUsers", model.Users{Users: users})
}

// FRIENDS

func (a *App) getFriends(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)
	// TODO
	//count, err, start, done := a.getStartCount(w, r)
	start, count := 0, 10

	friends, err := a.getFriendsData(start, count, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pending, err := a.getPendingFriendsData(start, count, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.Template.ExecuteTemplate(w, "friends", model.GetFriends{Friends: *friends, PendingFriends: *pending})
}

func (a *App) acceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

	vars := mux.Vars(r)
	friendUsername := vars["username"]

	friend, _ := a.Users.FindByUsername(friendUsername)

	// userOne is the user with the lowest ID
	userOne, userTwo := userID, friend.ID
	if userID > friend.ID {
		userOne, userTwo = friend.ID, userID
	}

	if err := a.Friendship.AcceptInvite(userOne, userTwo, userID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, "/index/friends", http.StatusFound)
}

func (a *App) declineInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

	vars := mux.Vars(r)
	friendUsername := vars["username"]

	friend, _ := a.Users.FindByUsername(friendUsername)

	// userOne is the user with the lowest ID
	userOne, userTwo := userID, friend.ID
	if userID > friend.ID {
		userOne, userTwo = friend.ID, userID
	}

	fmt.Println("IN DECLINE", userOne, userTwo)
	if err := a.Friendship.DeclineInvite(userOne, userTwo); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, "/index/friends", http.StatusFound)
}

func (a *App) addFriend(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+index+"/"+friends+"/add" {
		fmt.Println(r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	friendName := r.FormValue("username")

	// Check if username exists
	friend, err := a.Users.FindByUsername(friendName)
	if err != nil {
		message := fmt.Sprintf("There is no user: %v", friendName)
		respondWithError(w, http.StatusBadRequest, message)
		return
	}

	// userOne is the user with the lowest ID
	userOne, userTwo := userID, friend.ID
	if userID > friend.ID {
		userOne, userTwo = friend.ID, userID
	}

	friendship := &model.Friendship{
		UserOne:    userOne,
		UserTwo:    userTwo,
		ActionUser: userID,
	}

	if err := a.Friendship.Add(friendship); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, "/index/friends", http.StatusFound)
}

// HELPER
func (a *App) getFriendsData(start, count, userID int) (*model.Friends, error) {
	friendIDs, err := a.Friendship.Find(start, count, userID)
	if err != nil {
		return nil, err
	}

	friendNames, err := a.convertToUsername(friendIDs)
	if err != nil {
		return nil, err
	}

	return &model.Friends{Usernames: friendNames}, nil
}

func (a *App) getPendingFriendsData(start, count, userID int) (*model.Friends, error) {
	friendIDs, err := a.Friendship.FindPending(start, count, userID)
	if err != nil {
		return nil, err
	}

	friendNames, err := a.convertToUsername(friendIDs)

	return &model.Friends{Usernames: friendNames}, nil
}

func (a *App) convertToUsername(ids []int) ([]string, error) {
	usernames, err := a.Users.FindNamesByIDs(ids)
	if err != nil {
		return nil, err
	}
	return usernames, nil
}

//todo
func (a *App) getStartCount(w http.ResponseWriter, r *http.Request) (int, error, int, bool) {
	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil && r.FormValue("count") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request count parameter")
		return 0, nil, 0, true
	}
	start, err := strconv.Atoi(r.FormValue("start"))
	if err != nil && r.FormValue("start") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request start parameter")
		return 0, nil, 0, true
	}

	const (
		minOffset = 0
		minLimit  = 1
		maxLimit  = 10
	)

	start--
	if count > maxLimit || count < minLimit {
		count = maxLimit
	}
	if start < minOffset {
		start = minOffset
	}
	return count, err, start, false
}

// PAYMENT

// I want to pay 20lv for FOOD "Happy"
// Receive --> user_id, amount, categoryName, description
func (a *App) pay(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/index/"+pay {
		fmt.Println(r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		user := r.Context().Value("user").(*model.UserToken)
		userID, _ := strconv.Atoi(user.UserID)
		// Show balance
		balance, _ := a.Payment.CheckBalance(userID)

		// Show Expense Categories
		categories, _ := a.Categories.FindExpenses()

		// Show Friends
		friendIDs, _ := a.Friendship.Find(0, 100, userID) // TODO fix range
		friendUsernames, _ := a.convertToUsername(friendIDs)

		_ = a.Template.ExecuteTemplate(w, pay, model.PayTemplate{
			Balance:    balance,
			Categories: categories,
			Friends:    friendUsernames,
		})
	case "POST":
		userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

		if err := r.ParseForm(); err != nil {
			_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		amountS := r.FormValue("amount")
		amount, _ := strconv.Atoi(amountS)
		categoryName := r.FormValue("category")
		category, _ := a.Categories.FindByName(categoryName)
		description := r.FormValue("description")

		h := &model.History{
			UserID:      userID,
			Amount:      amount,
			CategoryID:  category.ID,
			Description: description,
		}

		err := a.Payment.Pay(h)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		http.Redirect(w, r, "/index/pay", http.StatusFound)
	default:
		_, _ = fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

// I giveMoneyTo George for "Bills"
// Receive --> <CreditorID> // DebtorID, Amount, Description
func (a *App) giveLoan(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	friendName := r.FormValue("to")
	friend, _ := a.Users.FindByUsername(friendName)
	amountS := r.FormValue("amount")
	amount, _ := strconv.Atoi(amountS)
	description := r.FormValue("description")

	var loan = "loan"
	loanC, _ := a.Categories.FindByName(loan)

	var debt = "debt"
	debtC, _ := a.Categories.FindByName(debt)

	t := &model.Transfer{
		CreditorID: userID,
		LoanID:     loanC.ID,
		DebtID:     debtC.ID,
		Loan: model.Loan{
			DebtorID:    friend.ID,
			Amount:      amount,
			Description: description,
		},
	}

	if err := a.Payment.GiveLoan(t); err != nil {
		msg := fmt.Sprintf("Error in giving money: %v", err.Error())
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	http.Redirect(w, r, "/index/pay", http.StatusFound)
}

// I want to split money with George for FOOD "Happy"
// Receive --> creditor_id, debtor_id, amount, categoryName, description
func (a *App) split(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	friendName := r.FormValue("to")
	friend, _ := a.Users.FindByUsername(friendName)
	amountS := r.FormValue("amount")
	amount, _ := strconv.Atoi(amountS)
	categoryName := r.FormValue("category")
	description := r.FormValue("description")

	var loan = "loan"
	loanC, _ := a.Categories.FindByName(loan)
	debtC, _ := a.Categories.FindByName(categoryName)

	t := &model.Transfer{
		CreditorID: userID,
		LoanID:     loanC.ID,
		DebtID:     debtC.ID,
		Loan: model.Loan{
			DebtorID:    friend.ID,
			Amount:      amount,
			Description: description,
		},
	}

	if err := a.Payment.Split(t); err != nil {
		msg := fmt.Sprintf("Error in splitting money: %v", err.Error())
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	http.Redirect(w, r, "/index/pay", http.StatusFound)
}

// I earn 1000lv from SALARY "Job"
// Receive --> user_id, amount, categoryName, description
func (a *App) earn(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/index/"+earn {
		fmt.Println(r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		user := r.Context().Value("user").(*model.UserToken)
		userID, _ := strconv.Atoi(user.UserID)
		// Show balance
		balance, _ := a.Payment.CheckBalance(userID)

		// Show Income Categories
		categories, _ := a.Categories.FindIncomes()

		// Show Friends
		friendIDs, _ := a.Friendship.Find(0, 100, userID) // TODO fix range
		friendUsernames, _ := a.convertToUsername(friendIDs)

		_ = a.Template.ExecuteTemplate(w, earn, model.PayTemplate{
			Balance:    balance,
			Categories: categories,
			Friends:    friendUsernames,
		})
	case "POST":
		userID, _ := strconv.Atoi(r.Context().Value("user").(*model.UserToken).UserID)

		if err := r.ParseForm(); err != nil {
			_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		amountS := r.FormValue("amount")
		amount, _ := strconv.Atoi(amountS)
		categoryName := r.FormValue("category")
		category, _ := a.Categories.FindByName(categoryName)
		description := r.FormValue("description")

		h := &model.History{
			UserID:      userID,
			Amount:      amount,
			CategoryID:  category.ID,
			Description: description,
		}

		err := a.Payment.Earn(h)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		http.Redirect(w, r, "/index/earn", http.StatusFound)
	default:
		_, _ = fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

// Receive --> DebtorID
// Return --> {StatusID, CreditorID, Amount, CategoryName, Description}
func (a *App) getDebts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/index/"+debts {
		fmt.Println(r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		user := r.Context().Value("user").(*model.UserToken)
		userID, _ := strconv.Atoi(user.UserID)

		// Show balance
		balance, _ := a.Payment.CheckBalance(userID)

		// Show debts:
		activeDebts, err := a.Payment.FindActiveDebts(userID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		ds := make([]model.DebtTemplate, 0, len(activeDebts))
		for _, d := range activeDebts {
			creditor, _ := a.Users.FindByID(d.CreditorID)
			ds = append(ds, model.DebtTemplate{
				StatusID:    d.StatusID,
				Creditor:    creditor.Username,
				Amount:      d.Amount,
				Description: d.Description,
			})
		}

		// Show pending debts:
		pendingDebts, err := a.Payment.FindPendingDebts(userID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		pds := make([]model.DebtTemplate, 0, len(pendingDebts))
		for _, pd := range pendingDebts {
			creditor, _ := a.Users.FindByID(pd.CreditorID)
			pds = append(pds, model.DebtTemplate{
				Creditor:    creditor.Username,
				Amount:      pd.Amount,
				Description: pd.Description,
			})
		}

		_ = a.Template.ExecuteTemplate(w, debts, model.DebtsTemplate{Active: ds, Pending: pds, Balance: balance})
		//case "POST":
	}
}

// I want to requestRepay => return my debt
// Receive --> debtID, amount
func (a *App) requestRepay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	debtIDS := vars["id"]
	debtID, _ := strconv.Atoi(debtIDS)

	if err := r.ParseForm(); err != nil {
		_, _ = fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	amountS := r.FormValue("amount")
	amount, _ := strconv.Atoi(amountS)

	err := a.Payment.RequestRepay(debtID, amount)
	if err != nil {
		fmt.Printf("Error requesting repay: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Invalid transfer")
		return
	}

	http.Redirect(w,r, "/"+index+"/"+debts, http.StatusFound)
}

//// Receive --> CreditorID
//// Return --> {DebtorID, Amount, Description}
//func (a *App) getLoans(w http.ResponseWriter, r *http.Request) {
//	// todo userid
//	var userID int
//
//	loans, err := a.Payment.FindActiveLoans(userID)
//	if err != nil {
//		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
//		return
//	}
//
//	respondWithJSON(w, http.StatusOK, loans)
//}

