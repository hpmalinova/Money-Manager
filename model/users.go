package model

import "github.com/dgrijalva/jwt-go"

type User struct {
	ID       int    `json:"id" validate:"numeric,gte=0"`
	Username string `json:"username" validate:"required,min=5,max=30"`
	Password string `json:"password,omitempty"`
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
