package model

import "github.com/dgrijalva/jwt-go"

type User struct {
	ID       int    `json:"id" validate:"numeric,gte=0"`
	Username string `json:"username" validate:"required,min=3,max=32"`
	Password string `json:"password,omitempty"` //todo better password
}

type UserToken struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Users struct {
	Users []User
}

type UserWallet struct {
	Username string
	Balance  int
}
