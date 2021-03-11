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
	index    = "index"
	logout   = "logout"

	users = "users"
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
	username := ctx.Value("user").(*model.UserToken).Username
	a.Template.ExecuteTemplate(w, index, username)
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

//func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
//	users, err := a.Users.Find(0, 200)
//	if err != nil {
//		respondWithError(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//	// remove user passwords
//	for i := range users {
//		users[i].Password = ""
//	}
//	respondWithJSON(w, http.StatusOK, users)
//}
func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	// TODO
	//count, err, start, done := a.getStartCount(w, r)
	//if done {
	//	return
	//}
	start,count := 0,10

	users, err := a.Users.Find(start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// remove user passwords
	for i := range users {
		users[i].Password = ""
	}

	a.Template.ExecuteTemplate(w,"showUsers", model.Users{Users: users})

	//respondWithJSON(w, http.StatusOK, users)
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
