package rest

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/hpmalinova/Money-Manager/model"
	"net/http"
)

func JwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t, err := r.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				// If the cookie is not set, return an unauthorized status
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// For any other type of error, return a bad request status
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token := t.Value
		if token == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing auth token")
			return
		}
		claims := &model.UserToken{}

		_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
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
