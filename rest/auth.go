package rest

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/hpmalinova/Money-Manager/model"
	"net/http"
	"strings"
)

func JwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const BEARER_SCHEMA = "Bearer "
		var header = r.Header.Get("Authorization") //Grab the token from the header

		var token string
		if header == "" || len(header) <= len(BEARER_SCHEMA) {
			respondWithError(w, http.StatusUnauthorized, "Missing auth token")
			return
		}
		token = header[len(BEARER_SCHEMA):]
		token = strings.TrimSpace(token)

		if token == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing auth token")
			return
		}
		claims := &model.UserToken{}

		_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil //todo env
		})

		if err != nil {
			respondWithError(w, http.StatusForbidden, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
