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
	register = "register"
	login    = "login"
)

func (a *App) initializeRoutes() {
	//a.Router.HandleFunc("/", hello).Methods(http.MethodGet)

	a.Router.HandleFunc("/"+register, a.registerHandler).Methods(http.MethodGet, http.MethodPost)
	a.Router.HandleFunc("/"+login, a.loginHandler).Methods(http.MethodGet, http.MethodPost)
	//a.Router.HandleFunc("/register", a.register).Methods(http.MethodGet)
	//a.Router.HandleFunc("/register", a.register).Methods(http.MethodPost)

	// todo delete from here
	a.Router.HandleFunc("/users", a.getUsers).Methods(http.MethodGet)

	//// Auth route
	//s := a.Router.PathPrefix("/home").Subrouter()
	//s.Use(JwtVerify) // Middleware
	//s.HandleFunc("/users", a.getUsers).Methods(http.MethodGet)
	//s.HandleFunc("/users/{id:[0-9]+}", a.getUser).Methods(http.MethodGet)
	//
	//s.HandleFunc("/friends", a.addFriend).Methods(http.MethodPost)
	//s.HandleFunc("/friends/{id:[0-9]+}", a.getFriends).Methods(http.MethodGet)
	//s.HandleFunc("/friends/{id:[0-9]+}/pending", a.getPending).Methods(http.MethodGet)
	//s.HandleFunc("/friends/{id:[0-9]+}/pending/{friend-id:[0-9]+}/accept", a.acceptInvite).Methods(http.MethodPut)  //todo uri + check put
	//s.HandleFunc("/friends/{id:[0-9]+}/pending/{friend-id:[0-9]+}/accept", a.declineInvite).Methods(http.MethodPut) //todo
	//
	//s.HandleFunc("/groups", a.addGroup).Methods(http.MethodPost)
	//s.HandleFunc("/groups/{id:[0-9]+}", a.getGroups).Methods(http.MethodGet)
	////s.HandleFunc("/groups/{id:[0-9]+}/split", a.payForGroup).Methods(http.MethodPost) // TODO split money between group members
	//
	//s.HandleFunc("/categories", a.getCategories).Methods(http.MethodGet)

}

func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+register {
		fmt.Println(r.URL.Path)
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
		name := r.FormValue("username")
		password := r.FormValue("password")
		fmt.Println(name, password)

		user := &model.User{Username: name, Password: password}

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
		name := r.FormValue("username")
		password := r.FormValue("password")
		fmt.Println(name, password)

		user := &model.UserLogin{Username: name, Password: password}
		err := a.Validator.Struct(user)
		if err != nil {
			errs := err.(validator.ValidationErrors)
			respondWithValidationError(errs.Translate(a.Translator), w)
			return
		}

		resp, err := a.checkCredentials(w, user.Username, user.Password)
		if err == nil {
			//_ = json.NewEncoder(w).Encode(resp)
			respondWithJSON(w, http.StatusOK, resp)
		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}
