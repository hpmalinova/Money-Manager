package rest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/hpmalinova/Money-Manager/model"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) checkCredentials(w http.ResponseWriter, username, password string) (map[string]string, error) {
	user, err := a.Users.FindByUsername(username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Username not found")
		return nil, err
	}
	expiresAt := time.Now().Add(time.Minute * 30).Unix()

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { //Password does not match!
		respondWithError(w, http.StatusUnauthorized, "Invalid login credentials. Please try again")
		return nil, err
	}

	claims := &model.UserToken{
		UserID:   strconv.Itoa(user.ID),
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, error := token.SignedString([]byte("secret"))
	if error != nil {
		fmt.Println(error)
	}

	var resp = map[string]string{"token": tokenString, "username": user.Username, "id": strconv.Itoa(user.ID)}
	return resp, nil
}

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

// TODO
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

// Future functions:
// TODO check history!
// TODO statistics

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

func (a *App) getCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := a.Categories.FindAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, categories)
}

func (a *App) getCategoryByName(categoryName string) *model.Category {
	fmt.Println("IN getCategoryByName", categoryName)
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
