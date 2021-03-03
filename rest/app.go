package rest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strconv"

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
	Users  contract.UserRepo

	Validator *validator.Validate
	Translator ut.Translator
}

func (a *App) Init(user, password, dbname string) {
	a.Users = repository.NewUserRepoMysql(user, password, dbname)

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
}

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
		UserID: string(user.ID),
		Username:   user.Username,
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
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(user); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

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
		minLimit = 1
		maxLimit = 10
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