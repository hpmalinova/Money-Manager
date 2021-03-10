package rest

import (
	"database/sql"
	"encoding/json"
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
	Payment    contract.PaymentRepo

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

// todo: getExpense/getIncome

func (a *App) getCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := a.Categories.FindAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, categories)
}

func (a *App) getCategoryByName(categoryName string) *model.Category {
	c, err := a.Categories.FindByName(categoryName)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

func (a *App) getCategoryByStatus(statusID int) string {
	cName, err := a.Payment.FindCategoryName(statusID)
	if err != nil {
		fmt.Println(err.Error())
	}
	return cName
}

// Debt //

// I want to pay 20lv for FOOD "Happy"
// Receive --> user_id, amount, categoryName, description
func (a *App) pay(w http.ResponseWriter, r *http.Request) {
	// todo userID and remove from Pay model
	payModel := &model.Pay{}
	err := json.NewDecoder(r.Body).Decode(payModel)

	if err != nil {
		fmt.Printf("Error paying : %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Find CategoryID
	category, err := a.Categories.FindByName(payModel.CategoryName)
	if err != nil {
		message := fmt.Sprintf("There is no category %s: %v", payModel.CategoryName, err.Error())
		respondWithError(w, http.StatusBadRequest, message)
	}

	h := &model.History{
		UserID:      payModel.UserID,
		Amount:      payModel.Amount,
		CategoryID:  category.ID,
		Description: payModel.Description,
	}

	err = a.Payment.Pay(h)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}
}

// I earn 1000lv from SALARY "Job"
// Receive --> user_id, amount, categoryName, description
func (a *App) earn(w http.ResponseWriter, r *http.Request) {
	// todo userID and remove from Pay model
	payModel := &model.Pay{}
	err := json.NewDecoder(r.Body).Decode(payModel)

	if err != nil {
		fmt.Printf("Error paying : %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Find CategoryID
	category, err := a.Categories.FindByName(payModel.CategoryName)
	if err != nil {
		message := fmt.Sprintf("There is no category %s: %v", payModel.CategoryName, err.Error())
		respondWithError(w, http.StatusBadRequest, message)
	}

	h := &model.History{
		UserID:      payModel.UserID,
		Amount:      payModel.Amount,
		CategoryID:  category.ID,
		Description: payModel.Description,
	}

	err = a.Payment.Earn(h)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}
}

// I giveMoneyTo George for "Bills"
// Receive --> <CreditorID> // DebtorID, Amount, Description
func (a *App) giveLoan(w http.ResponseWriter, r *http.Request) {
	// TODO get user id
	var userID int

	gm := model.Loan{}
	err := json.NewDecoder(r.Body).Decode(gm)
	if err != nil {
		fmt.Printf("Error in giving money : %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var loan = "Loan"
	loanC, err := a.Categories.FindByName(loan)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", loan)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	var debt = "Debt"
	debtC, err := a.Categories.FindByName(debt)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", debt)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	t := &model.Transfer{
		CreditorID: userID,
		LoanID:     loanC.ID,
		DebtID:     debtC.ID,
		Loan:       gm,
	}

	if err = a.Payment.GiveLoan(t); err != nil {
		msg := fmt.Sprintf("Error in giving money: %v", err.Error())
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
}

// I want to split money with George for FOOD "Happy"
// Receive --> creditor_id, debtor_id, amount, categoryName, description
func (a *App) split(w http.ResponseWriter, r *http.Request) {
	var userID int // TODO

	g := model.Give{}
	err := json.NewDecoder(r.Body).Decode(g)
	if err != nil {
		fmt.Printf("Error splitting money: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var loan = "Loan"
	loanC, err := a.Categories.FindByName(loan)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", loan)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	debtC, err := a.Categories.FindByName(g.CategoryName)
	if err != nil {
		msg := fmt.Sprintf("No category: %s", g.CategoryName)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}

	t := &model.Transfer{
		CreditorID: userID,
		LoanID:     loanC.ID,
		DebtID:     debtC.ID,
		Loan:       g.Loan,
	}

	if err := a.Payment.Split(t); err != nil {
		msg := fmt.Sprintf("Error in splitting money: %v", err.Error())
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
}

// Receive --> DebtorID
// Return --> {StatusID, CreditorID, Amount, CategoryName, Description}
func (a *App) getDebts(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	debts, err := a.Payment.FindActiveDebts(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, debts)
}

// Receive --> CreditorID
// Return --> {DebtorID, Amount, Description}
func (a *App) getLoans(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	loans, err := a.Payment.FindActiveLoans(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, loans)
}

// I want to requestRepay => return my debt
// Receive --> debtID, amount
func (a *App) requestRepay(w http.ResponseWriter, r *http.Request) {
	rr := &model.RepayRequest{}
	err := json.NewDecoder(r.Body).Decode(rr)

	if err != nil {
		fmt.Printf("Error requesting repay: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = a.Payment.RequestRepay(rr.DebtID, rr.Amount)
	if err != nil {
		fmt.Printf("Error requesting repay: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Invalid transfer")
		return
	}
}

// The user waits for Peter to accept his payment
// Receive --> debtorID
// Return  --> {creditor, amount, description}
func (a *App) getPendingDebts(w http.ResponseWriter, r *http.Request) {
	// todo userid
	var userID int

	debts, err := a.Payment.FindPendingDebts(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, debts)
}

// Peter has sent you a repay request.
// Will you (as creditor) accept or decline it?
// Receive --> creditorID
// Return  --> {debtorID, pendingAmount, description, statusID}
func (a *App) getPendingRequests(w http.ResponseWriter, r *http.Request) {
	// todo userID == creditorID
	var userID int

	loans, err := a.Payment.FindPendingRequests(userID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	respondWithJSON(w, http.StatusOK, loans)
}

// Peter has sent you a repay request. You acceptPayment.
// Receive --> statusID
func (a *App) acceptPayment(w http.ResponseWriter, r *http.Request) {
	var statusID int
	err := json.NewDecoder(r.Body).Decode(&statusID)
	if err != nil {
		fmt.Printf("Error accepting payment: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	expenseC := a.getCategoryByName(a.getCategoryByStatus(statusID))
	repayC := a.getCategoryByName("Repay")

	//var repay = "Repay"
	//repayC, err := a.Categories.FindByName(repay)
	//if err != nil {
	//	msg := fmt.Sprintf("No category: %s", repay)
	//	respondWithError(w, http.StatusInternalServerError, msg)
	//	return
	//}

	am := &model.Accept{StatusID: statusID, RepayC: *repayC, ExpenseC: *expenseC}

	if err = a.Payment.AcceptPayment(am); err!=nil{
		msg := fmt.Sprintf("Error accepting payment: %v", err.Error())
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
}
