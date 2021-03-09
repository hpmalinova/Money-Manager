package rest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strconv"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"

	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"

	"github.com/gorilla/mux"

	"github.com/hpmalinova/Money-Manager/contract"
	"github.com/hpmalinova/Money-Manager/model"
	"github.com/hpmalinova/Money-Manager/repository"

	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"
)

type App struct {
	Router *mux.Router

	Users      contract.UserRepo
	Friendship contract.FriendshipRepo
	Groups     contract.GroupRepo
	Categories contract.CategoryRepo
	History    contract.HistoryRepo
	Debt       contract.DebtRepo

	Validator  *validator.Validate
	Translator ut.Translator
}

func (a *App) Init(user, password, dbname string) {
	// db=sqlopen
	// newrepo(&db) --> repo.db = db
	a.Users = repository.NewUserRepoMysql(user, password, dbname) // TODO one db connection?
	a.Friendship = repository.NewFriendRepoMysql(user, password, dbname)
	a.Groups = repository.NewGroupRepoMysql(user, password, dbname)
	a.Categories = repository.NewCategoryRepoMysql(user, password, dbname)
	a.History = repository.NewHistoryRepoMysql(user, password, dbname)
	a.Debt = repository.NewDebtRepoMysql(user, password, dbname)

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
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/login", a.login).Methods(http.MethodPost)
	a.Router.HandleFunc("/register", a.register).Methods(http.MethodPost)

	// Auth route
	s := a.Router.PathPrefix("/home").Subrouter()
	s.Use(JwtVerify)
	s.HandleFunc("/users", a.getUsers).Methods(http.MethodGet)
	s.HandleFunc("/users/{id:[0-9]+}", a.getUser).Methods(http.MethodGet)

	s.HandleFunc("/friends", a.addFriend).Methods(http.MethodPost)
	s.HandleFunc("/friends/{id:[0-9]+}", a.getFriends).Methods(http.MethodGet)
	s.HandleFunc("/friends/{id:[0-9]+}/pending", a.getPending).Methods(http.MethodGet)
	s.HandleFunc("/friends/{id:[0-9]+}/pending/{friend-id:[0-9]+}/accept", a.acceptInvite).Methods(http.MethodPut)  //todo uri + check put
	s.HandleFunc("/friends/{id:[0-9]+}/pending/{friend-id:[0-9]+}/accept", a.declineInvite).Methods(http.MethodPut) //todo

	s.HandleFunc("/groups", a.addGroup).Methods(http.MethodPost)
	s.HandleFunc("/groups/{id:[0-9]+}", a.getGroups).Methods(http.MethodGet)
	//s.HandleFunc("/groups/{id:[0-9]+}/split", a.payForGroup).Methods(http.MethodPost) // TODO split money between group members

	s.HandleFunc("/categories", a.getCategories).Methods(http.MethodGet)

}

// Users //
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	userCredentials := &model.UserLogin{}
	err := json.NewDecoder(r.Body).Decode(userCredentials)
	if err != nil {
		fmt.Printf("Error logging user %v: %v", userCredentials, err)
		var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	resp, err := a.checkCredentials(w, userCredentials.Username, userCredentials.Password)
	if err == nil {
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (a *App) checkCredentials(w http.ResponseWriter, username, password string) (map[string]interface{}, error) {
	user, err := a.Users.FindByUsername(username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Username not found")
		return nil, err
	}
	expiresAt := time.Now().Add(time.Minute * 10).Unix()

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { //Password does not match!
		respondWithError(w, http.StatusUnauthorized, "Invalid login credentials. Please try again")
		return nil, err
	}

	claims := &model.UserToken{
		UserID:   string(user.ID),
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			//Issuer:    "test",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, error := token.SignedString([]byte("secret"))
	if error != nil {
		// TODO respond with error?
		fmt.Println(error)
	}

	var resp = map[string]interface{}{"status": false, "message": "logged in"}
	resp["token"] = tokenString //Store the token in the response
	// remove user password
	user.Password = ""

	resp["user"] = user
	return resp, nil
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	user := &model.User{}

	// r.Body: {"username":"peter", "password": "123"}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	//decoder := json.NewDecoder(r.Body)
	//if err := decoder.Decode(user); err != nil {
	//	respondWithError(w, http.StatusBadRequest, "Invalid request payload")
	//	return
	//}

	// Validate User struct
	err := a.Validator.Struct(user)
	if err != nil {
		// translate all error at once
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
	// remove user password
	user.Password = ""

	respondWithJSON(w, http.StatusCreated, user)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil && r.FormValue("count") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request count parameter")
		return
	}
	start, err := strconv.Atoi(r.FormValue("start"))
	if err != nil && r.FormValue("start") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request start parameter")
		return
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
	users, err := a.Users.Find(start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// remove user passwords
	for i := range users {
		users[i].Password = ""
	}
	respondWithJSON(w, http.StatusOK, users)
}

func (a *App) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user *model.User
	if user, err = a.Users.FindByID(id); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "User not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	// remove user password
	user.Password = ""

	respondWithJSON(w, http.StatusOK, user)
}

// Friendship //

func (a *App) addFriend(w http.ResponseWriter, r *http.Request) {
	addFriendModel := &model.AddFriend{}
	err := json.NewDecoder(r.Body).Decode(addFriendModel)

	if err != nil {
		fmt.Printf("Error adding friend %v: %v", addFriendModel.FriendName, err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		//var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		//_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Create friendship model:

	user, err := a.Users.FindByUsername(addFriendModel.FriendName)
	if err != nil {
		message := fmt.Sprintf("There is no user: %v", addFriendModel.FriendName)
		respondWithError(w, http.StatusBadRequest, message)
	}

	userOne, userTwo := addFriendModel.ActionUserID, user.ID

	// userOne is the user with the lowest ID
	if addFriendModel.ActionUserID > user.ID {
		userOne, userTwo = user.ID, addFriendModel.ActionUserID
	}

	friendship := &model.Friendship{
		UserOne:    userOne,
		UserTwo:    userTwo,
		ActionUser: addFriendModel.ActionUserID,
	}

	if err := a.Friendship.Add(friendship); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *App) getFriends(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil && r.FormValue("count") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request count parameter")
		return
	}
	start, err := strconv.Atoi(r.FormValue("start"))
	if err != nil && r.FormValue("start") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request start parameter")
		return
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

	// TODO
	friendIDs, err := a.Friendship.Find(start, count, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// TODO convert to usernames
	//a.Users.
	respondWithJSON(w, http.StatusOK, friendIDs)
}

func (a *App) getPending(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil && r.FormValue("count") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request count parameter")
		return
	}
	start, err := strconv.Atoi(r.FormValue("start"))
	if err != nil && r.FormValue("start") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request start parameter")
		return
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

	/////////////////////////////////////////////////////////////////

	// TODO
	friendIDs, err := a.Friendship.FindPending(start, count, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// TODO convert to usernames
	//a.Users.
	respondWithJSON(w, http.StatusOK, friendIDs)
}

func (a *App) acceptInvite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	friendID, err := strconv.Atoi(vars["friend-id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid friend ID")
		return
	}

	userOne, userTwo := userID, friendID

	// userOne is the user with the lowest ID
	if userID > friendID {
		userOne, userTwo = friendID, userID
	}

	if err := a.Friendship.AcceptInvite(userOne, userTwo, userID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (a *App) declineInvite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	friendID, err := strconv.Atoi(vars["friend-id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid friend ID")
		return
	}

	userOne, userTwo := userID, friendID

	// userOne is the user with the lowest ID
	if userID > friendID {
		userOne, userTwo = friendID, userID
	}

	if err := a.Friendship.DeclineInvite(userOne, userTwo, userID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// Groups //

// Receives name + participantNames[]
// A user cannot participate in two groups with the same name
// TODO Redirect to page /group/{id}
func (a *App) addGroup(w http.ResponseWriter, r *http.Request) {
	// todo add the logged user id?
	// var1: as query parameter /addGroup/{my_id} --> da ne moje drug acc da go otvarq
	// var2: creatorID in the createGroup model

	createGroupModel := &model.CreateGroup{}
	err := json.NewDecoder(r.Body).Decode(createGroupModel)

	if err != nil {
		fmt.Printf("Error creating group %v: %v", createGroupModel.Name, err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		//var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		//_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// todo convert usernames to uids
	participants := []int{}

	if err := a.Groups.Create(createGroupModel.Name, participants); err != nil {
		prefix := "Bad Request: "
		if strings.HasPrefix(err.Error(), prefix) {
			respondWithError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), prefix))
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *App) getGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// todo in function? repeated code
	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil && r.FormValue("count") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request count parameter")
		return
	}
	start, err := strconv.Atoi(r.FormValue("start"))
	if err != nil && r.FormValue("start") != "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request start parameter")
		return
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

	groups, err := a.Groups.Find(start, count, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, groups)
}

// Categories //

// todo add getExpense/getIncome

func (a *App) getCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := a.Categories.FindAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, categories)
}

// Debt //

// I want to pay 20lv for FOOD "Happy"
// Receive --> user_id, amount, categoryName, description
func (a *App) pay(w http.ResponseWriter, r *http.Request) {
	payModel := &model.Pay{}
	err := json.NewDecoder(r.Body).Decode(payModel)

	if err != nil {
		fmt.Printf("Error paying : %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		//var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		//_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Find CategoryID
	category, err := a.Categories.FindByName(payModel.CategoryName)
	if err != nil {
		message := fmt.Sprintf("There is no category %s: %v", payModel.CategoryName, err.Error())
		respondWithError(w, http.StatusBadRequest, message)
	}

	historyModel := &model.History{
		UserID:      payModel.UserID,
		Amount:      payModel.Amount,
		CategoryID:  category.ID,
		Description: payModel.Description,
	}

	statusCode, err := a.payHelp(historyModel)
	if err != nil {
		respondWithError(w, statusCode, err.Error())
	}

	//// TODO wallet
	//// If enough money in wallet => pay and remove money
	//err = a.History.Pay(historyModel)
	//if err != nil {
	//	message := fmt.Sprintf("Unsuccessful payment: %v", err.Error())
	//	respondWithError(w, http.StatusBadRequest, message)
	//}
}

func (a *App) payHelp(historyModel *model.History) (int, error) {
	// TODO wallet
	// If enough money in wallet => pay and remove money
	err := a.History.Pay(historyModel)
	if err != nil {
		message := fmt.Sprintf("Unsuccessful payment: %v", err.Error())
		return http.StatusBadRequest, errors.New(message)
	}
	return http.StatusOK, nil
}

// todo check if friend
// I want to giveLoan to George
// Receive --> user_id, debtor_id, amount, description
func (a *App) giveLoan(w http.ResponseWriter, r *http.Request) {
	// todo get user id
	var userID int
	giveLoan := &model.GiveLoan{}
	err := json.NewDecoder(r.Body).Decode(giveLoan)

	if err != nil {
		fmt.Printf("Error giving loan : %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cLoan, err := a.Categories.FindByName(loan)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", loan)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	loanH := model.LoanHistory{
		DebtorID: giveLoan.DebtorID,
		History: model.History{
			UserID:      userID,
			Amount:      giveLoan.Amount,
			CategoryID:  cLoan.ID,
			Description: giveLoan.Description,
		},
	}

	dl := model.DebtAndLoan{
		CreditorID:   userID,
		DebtorID:     giveLoan.DebtorID,
		Amount:       giveLoan.Amount,
		CategoryName: cLoan.Name,
		Description:  giveLoan.Description,
	}

	statusCode, err := a.giveLoanHelp(loanH, dl)
	if err != nil {
		respondWithError(w, statusCode, err.Error())
	}
}

const (
	loan = "Loan"
)

func (a *App) giveLoanHelp(loanH model.LoanHistory, dl model.DebtAndLoan) (int, error) {
	// TODO Remove money from wallet

	// Add to expenses
	err := a.History.GiveLoan(&loanH)
	if err != nil {
		msg := fmt.Sprintf("Error in giving loan: %s", err)
		//respondWithError(w, http.StatusInternalServerError , msg) // todo delete
		return http.StatusInternalServerError, errors.New(msg)
	}

	// Add to debt
	err = a.Debt.Add(&dl)
	if err != nil {
		msg := fmt.Sprintf("Error in creating debt: %s", err)
		//respondWithError(w, http.StatusInternalServerError, msg) // TODO delete
		return http.StatusInternalServerError, errors.New(msg)
	}

	return http.StatusOK, nil
}

// I want to split money with George
// Receive --> user_id, debtor_id, amount, categoryName, description
func (a *App) split(w http.ResponseWriter, r *http.Request) {
	var userID int // TODO

	split := &model.Split{}
	err := json.NewDecoder(r.Body).Decode(split)
	if err != nil {
		fmt.Printf("Error splitting money: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		//var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		//_ = json.NewEncoder(w).Encode(resp)
		return
	}

	halfAmount := split.Amount / 2 // TODO check round

	// Pay my part of the split
	category, _ := a.Categories.FindByName(split.CategoryName)

	historyModel := &model.History{
		UserID:      userID,
		Amount:      halfAmount,
		CategoryID:  category.ID,
		Description: split.Description,
	}

	statusCode, err := a.payHelp(historyModel)
	if err != nil {
		respondWithError(w, statusCode, err.Error())
	}

	// Give money to George
	cLoan, err := a.Categories.FindByName(loan)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", loan)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	loanH := model.LoanHistory{
		DebtorID: split.DebtorID,
		History: model.History{
			UserID:      userID,
			Amount:      halfAmount,
			CategoryID:  cLoan.ID,
			Description: split.Description,
		},
	}

	dl := model.DebtAndLoan{
		CreditorID:   userID,
		DebtorID:     split.DebtorID,
		Amount:       halfAmount,
		CategoryName: cLoan.Name,
		Description:  split.Description,
	}

	statusCode, err = a.giveLoanHelp(loanH, dl)
	if err != nil {
		respondWithError(w, statusCode, err.Error())
	}
}

func (a *App) getDebts(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	debts, err := a.Debt.FindActiveDebts(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, debts)
}

func (a *App) getLoans(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	debts, err := a.Debt.FindActiveLoans(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, debts)
}

// Receive --> debtID, amount
func (a *App) sendRepayRequest(w http.ResponseWriter, r *http.Request) {
	// todo check if debtorID==userID
	rr := &model.RepayRequest{}
	err := json.NewDecoder(r.Body).Decode(rr)

	if err != nil {
		fmt.Printf("Error repaying debt: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = a.Debt.RequestPaymentConfirmation(rr.DebtID, rr.Amount)
	if err != nil {
		fmt.Printf("Error repaying debt: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid transfer")
		return
	}
}

// The user waits for Peter to accept his payment
// Receive --> debtorID
// Return  --> {creditor, amount, description}
func (a *App) getPendingDebts(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	debts, err := a.Debt.FindPendingDebts(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, debts)
}

// Peter has sent you a repay request.
// Will you (as creditor) accept or decline it?
// Receive --> creditorID
// Return  --> {debtorID, amount, description, statusID}
func (a *App) getPendingRequests(w http.ResponseWriter, r *http.Request) {
	// todo userID == creditorID
	var userID int

	loans, err := a.Debt.FindPendingRequests(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, loans)
}

// Receive --> {debtorID, amount, description, statusID}
func (a *App) acceptPayment(w http.ResponseWriter, r *http.Request) {
	l := &model.Loan{}
	err := json.NewDecoder(r.Body).Decode(l)
	if err != nil {
		fmt.Printf("Error accepting request: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	repayAmount, err := a.Debt.AcceptPayment(l.StatusID, l.Amount)
	if err != nil {
		fmt.Printf("Error declining request: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Invalid request payload")
		return
	}

	// todo remove that amount from debtor's wallet
	fmt.Println(repayAmount)
}

// Receive --> {statusID}
func (a *App) declinePayment(w http.ResponseWriter, r *http.Request) {
	var statusID int
	err := json.NewDecoder(r.Body).Decode(statusID)
	if err != nil {
		fmt.Printf("Error declining request: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := a.Debt.DeclinePayment(statusID); err != nil {
		fmt.Printf("Error declining request: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Invalid request payload")
		return
	}
}